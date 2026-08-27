import { afterEach, describe, expect, it } from "vitest";
import { ApiError, SeoClient, parseSseJsonMessages, setFetch, since } from "../src/client.js";

type Call = { url: string; init: RequestInit | undefined };

function fakeFetch(status: number, body: unknown, headers: Record<string, string> = { "content-type": "application/json" }) {
  const calls: Call[] = [];
  setFetch(async (url, init) => {
    calls.push({ url, init });
    const text = typeof body === "string" ? body : JSON.stringify(body);
    return new Response(text === "" ? null : text, { status, headers });
  });
  return calls;
}

afterEach(() => {
  setFetch((input, init) => fetch(input, init));
});

describe("SeoClient", () => {
  const client = new SeoClient({ baseUrl: "http://localhost:3011/", token: "seo_test" });

  it("sends bearer auth and encodes the slug and query", async () => {
    const calls = fakeFetch(200, { rows: [], summary: {} });
    await client.ranks("my project", { since: "7d", set: "guarantee" });
    expect(calls[0]?.url).toBe("http://localhost:3011/api/v1/projects/my%20project/ranks?since=7d&set=guarantee");
    expect(calls[0]?.init?.method).toBe("GET");
    expect(calls[0]?.init?.headers).toMatchObject({ authorization: "Bearer seo_test", accept: "application/json" });
    expect(calls[0]?.init?.body).toBeUndefined();
  });

  it("omits undefined query params", async () => {
    const calls = fakeFetch(200, []);
    await client.listKeywords("demo", { set: undefined });
    expect(calls[0]?.url).toBe("http://localhost:3011/api/v1/projects/demo/keywords");
  });

  it("posts JSON bodies", async () => {
    const calls = fakeFetch(200, { estimate: { cost: 0.012 } });
    const result = await client.research("demo", { seeds: ["seo tools"], limit: 20, max_kd: 25, dry_run: true });
    expect(calls[0]?.url).toBe("http://localhost:3011/api/v1/projects/demo/research");
    expect(calls[0]?.init?.method).toBe("POST");
    expect(calls[0]?.init?.headers).toMatchObject({ "content-type": "application/json" });
    expect(JSON.parse(String(calls[0]?.init?.body))).toEqual({ seeds: ["seo tools"], limit: 20, max_kd: 25, dry_run: true });
    expect(result).toEqual({ estimate: { cost: 0.012 } });
  });

  it("uses PATCH and DELETE for keyword updates", async () => {
    const calls = fakeFetch(200, {});
    await client.updateKeyword("demo", 7, { track: "bofu" });
    await client.deleteKeyword("demo", 7);
    expect(calls.map((call) => [call.init?.method, call.url])).toEqual([
      ["PATCH", "http://localhost:3011/api/v1/projects/demo/keywords/7"],
      ["DELETE", "http://localhost:3011/api/v1/projects/demo/keywords/7"],
    ]);
  });

  it("surfaces {error, code} responses as ApiError", async () => {
    fakeFetch(402, { error: "insufficient balance", code: "spend_blocked" });
    const failure = await client.serp("demo", { keyword: "x" }).catch((error: unknown) => error);
    expect(failure).toBeInstanceOf(ApiError);
    expect(failure).toMatchObject({ message: "insufficient balance", code: "spend_blocked", status: 402 });
  });

  it("falls back to the HTTP status when the error body is not JSON", async () => {
    fakeFetch(500, "<html>boom</html>", { "content-type": "text/html" });
    const failure = await client.whoami().catch((error: unknown) => error);
    expect(failure).toMatchObject({ code: "http_500", status: 500 });
  });

  it("wraps network failures", async () => {
    setFetch(async () => {
      throw new Error("ECONNREFUSED");
    });
    const failure = await client.whoami().catch((error: unknown) => error);
    expect(failure).toMatchObject({ code: "network_error", status: 0 });
    expect((failure as Error).message).toContain("http://localhost:3011");
  });

  it("returns null for empty bodies", async () => {
    fakeFetch(204, "");
    expect(await client.deleteKeyword("demo", 1)).toBeNull();
  });

  it("speaks JSON-RPC to /mcp and remembers the session id", async () => {
    const calls: Call[] = [];
    let requestCount = 0;
    setFetch(async (url, init) => {
      calls.push({ url, init });
      requestCount += 1;
      const body = JSON.parse(String(init?.body)) as { id?: number };
      const headers: Record<string, string> = { "content-type": "application/json" };
      if (requestCount === 1) headers["mcp-session-id"] = "sess-1";
      return new Response(JSON.stringify({ jsonrpc: "2.0", id: body.id, result: { tools: [{ name: "get_ranks" }] } }), { status: 200, headers });
    });
    const first = await client.mcpRequest("initialize", { protocolVersion: "2025-06-18" });
    await client.mcpRequest("tools/list");
    expect(first).toEqual({ tools: [{ name: "get_ranks" }] });
    expect(calls[0]?.url).toBe("http://localhost:3011/mcp");
    expect(calls[0]?.init?.headers).toMatchObject({ accept: "application/json, text/event-stream", authorization: "Bearer seo_test" });
    expect(calls[1]?.init?.headers).toMatchObject({ "mcp-session-id": "sess-1" });
  });

  it("reads JSON-RPC results from an SSE body and raises JSON-RPC errors", async () => {
    let requestCount = 0;
    setFetch(async (_url, init) => {
      requestCount += 1;
      const body = JSON.parse(String(init?.body)) as { id?: number };
      if (requestCount === 1) {
        const sse = `event: message\ndata: ${JSON.stringify({ jsonrpc: "2.0", id: body.id, result: { ok: true } })}\n\n`;
        return new Response(sse, { status: 200, headers: { "content-type": "text/event-stream" } });
      }
      return new Response(JSON.stringify({ jsonrpc: "2.0", id: body.id, error: { code: -32602, message: "unknown tool" } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    });
    expect(await client.mcpRequest("ping")).toEqual({ ok: true });
    const failure = await client.mcpRequest("tools/call", { name: "nope" }).catch((error: unknown) => error);
    expect(failure).toMatchObject({ message: "unknown tool", code: "mcp_-32602" });
  });

  it("parses multi-line SSE data blocks", () => {
    const text = 'data: {"jsonrpc":"2.0",\ndata: "id":1,"result":{}}\n\n: keepalive\n\ndata: {"jsonrpc":"2.0","id":2,"result":{"a":1}}\n';
    expect(parseSseJsonMessages(text).map((message) => message.id)).toEqual([1, 2]);
  });
});

describe("since", () => {
  const now = new Date("2026-08-27T12:00:00Z");

  it("accepts day windows", () => {
    expect(since("7d", now)).toEqual({ query: "7d", from: new Date("2026-08-20T12:00:00Z") });
    expect(since(" 30d ", now).query).toBe("30d");
  });

  it("accepts ISO dates", () => {
    expect(since("2026-08-01", now)).toEqual({ query: "2026-08-01", from: new Date("2026-08-01T00:00:00Z") });
  });

  it("rejects everything else", () => {
    expect(() => since("3w", now)).toThrow(/invalid --since/);
    expect(() => since("2026-02-30", now)).toThrow(/invalid date/);
    expect(() => since("yesterday", now)).toThrow(/invalid --since/);
  });
});
