import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { setFetch } from "../src/client.js";
import { saveStoredConfig } from "../src/config.js";
import { setWriter } from "../src/output.js";
import { buildProgram } from "../src/program.js";

type Call = { url: string; method: string | undefined; headers: Record<string, string>; body: unknown };

let directory: string;
let output: string;
let calls: Call[];

function respondWith(status: number, body: unknown) {
  setFetch(async (url, init) => {
    const headers = (init?.headers ?? {}) as Record<string, string>;
    calls.push({ url, method: init?.method, headers, body: init?.body ? JSON.parse(String(init.body)) : undefined });
    return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
  });
}

async function run(...args: string[]): Promise<void> {
  const program = buildProgram();
  program.exitOverride();
  await program.parseAsync(args, { from: "user" });
}

beforeEach(() => {
  directory = mkdtempSync(join(tmpdir(), "seo-cli-commands-"));
  process.env.SEO_CONFIG_PATH = join(directory, "config.json");
  saveStoredConfig({ baseUrl: "http://backend.test", token: "seo_cfg", project: "demo" }, process.env.SEO_CONFIG_PATH);
  output = "";
  calls = [];
  setWriter((text) => {
    output += text;
  });
});

afterEach(() => {
  delete process.env.SEO_CONFIG_PATH;
  delete process.env.SEO_PROJECT;
  setFetch((input, init) => fetch(input, init));
  setWriter((text) => process.stdout.write(text));
  rmSync(directory, { recursive: true, force: true });
});

const ranksResponse = {
  rows: [
    { keyword: "seo cli", position: 3, previous: 7, delta: 4, band_change: "entered_top10", url: "/blog/seo-cli", path_match: true },
    { keyword: "rank tracker with api for coding agents", position: 18, previous: 15, delta: -3, band_change: null, url: null, path_match: false },
    { keyword: "new keyword", position: null, previous: null, delta: null, band_change: null, url: null, path_match: false },
  ],
  summary: { top10: 1, top20: 2, top100: 2, unranked: 1, avg_position: 10.5, floor_target: 3, floor_met: false },
};

describe("ranks", () => {
  it("prints a keyword/pos/prev/delta/band/url table and the summary line", async () => {
    respondWith(200, ranksResponse);
    await run("ranks", "--since", "7d");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/ranks?since=7d&set=guarantee");
    expect(calls[0]?.headers.authorization).toBe("Bearer seo_cfg");
    expect(output).toBe(
      [
        "keyword                                  pos  prev  delta  band           url",
        "seo cli                                    3     7     +4  entered_top10  /blog/seo-cli",
        "rank tracker with api for coding agents   18    15     -3  -              -",
        "new keyword                                -     -      -  -              -",
        "top10=1 top20=2 top100=2 unranked=1 avg_position=10.5 floor_target=3 floor_met=no",
        "",
      ].join("\n"),
    );
  });

  it("passes --set and an ISO --since through and honours --project and SEO_PROJECT", async () => {
    respondWith(200, ranksResponse);
    process.env.SEO_PROJECT = "env-project";
    await run("ranks", "--since", "2026-08-01", "--set", "measure");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/env-project/ranks?since=2026-08-01&set=measure");
    await run("ranks", "--project", "flag-project");
    expect(calls[1]?.url).toBe("http://backend.test/api/v1/projects/flag-project/ranks?since=7d&set=guarantee");
  });

  it("prints raw JSON with --json", async () => {
    respondWith(200, ranksResponse);
    await run("ranks", "--json");
    expect(JSON.parse(output)).toEqual(ranksResponse);
  });

  it("surfaces API errors", async () => {
    respondWith(404, { error: "project not found", code: "not_found" });
    await expect(run("ranks")).rejects.toMatchObject({ message: "project not found", code: "not_found", status: 404 });
    expect(output).toBe("");
  });
});

describe("research --dry-run", () => {
  it("sends dry_run and prints the estimate", async () => {
    respondWith(200, { estimate: { cost: 0.0132, keywords_count: 0, seeds_count: 2 } });
    await run("research", "seo cli", "rank tracker", "--limit", "20", "--max-kd", "25", "--dry-run");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/research");
    expect(calls[0]?.method).toBe("POST");
    expect(calls[0]?.body).toEqual({ seeds: ["seo cli", "rank tracker"], limit: 20, max_kd: 25, dry_run: true });
    expect(output).toBe("estimate $0.0132 (keywords_count=0 seeds_count=2) — dry run, nothing spent\n");
  });

  it("omits dry_run and prints the table without it", async () => {
    respondWith(200, { keywords: [{ keyword: "seo cli", volume: 210, kd: 12, cpc: 1.5, intent: "commercial", yoy: 0.2 }], cost: 0.012 });
    await run("research", "seo cli");
    expect(calls[0]?.body).toEqual({ seeds: ["seo cli"] });
    expect(output).toBe("keyword  volume  kd  cpc  intent      yoy\nseo cli     210  12  1.5  commercial  0.2\ncost $0.012\n");
  });

  it("is what `seo estimate research ...` re-invokes", async () => {
    respondWith(200, { estimate: { cost: 0.024 } });
    await run("estimate", "research", "seo cli", "--limit", "10", "--project", "other");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/other/research");
    expect(calls[0]?.body).toEqual({ seeds: ["seo cli"], limit: 10, dry_run: true });
    expect(output).toBe("estimate $0.024 — dry run, nothing spent\n");
  });

  it("refuses to estimate a free command", async () => {
    await expect(run("estimate", "ranks")).rejects.toThrow(/paid commands/);
  });
});

describe("locked sets", () => {
  it("sends locked on keywords add and prompts add", async () => {
    respondWith(200, {});
    await run("keywords", "add", "seo cli", "--set", "guarantee", "--track", "bofu", "--path", "/seo-cli", "--locked");
    expect(calls[0]?.body).toEqual({
      keywords: [{ keyword: "seo cli", track: "bofu", target_path: "/seo-cli", set_name: "guarantee", locked: true }],
    });
    await run("prompts", "add", "best seo cli", "--locked");
    expect(calls[1]?.body).toEqual({ prompts: [{ text: "best seo cli", locked: true }] });
  });

  it("renders a locked column", async () => {
    respondWith(200, { keywords: [{ id: 1, keyword: "seo cli", set_name: "guarantee", locked: true }] });
    await run("keywords", "list");
    expect(output.split("\n")[0]).toMatch(/^id  keyword  set        track  path  volume  kd  locked$/);
    expect(output).toContain("yes");
  });
});
