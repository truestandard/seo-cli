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

describe("domain", () => {
  it("sends domain, limit and dry_run and prints the estimate", async () => {
    respondWith(200, { estimate: { cost: 0.04092, domain: "rival.example", limit: 20 } });
    await run("domain", "rival.example", "--limit", "20", "--dry-run");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/domain_overview");
    expect(calls[0]?.body).toEqual({ domain: "rival.example", limit: 20, dry_run: true });
    expect(output).toBe("estimate $0.0409 (domain=rival.example limit=20) — dry run, nothing spent\n");
  });

  it("renders traffic, keywords, top keywords, top pages and the cache line", async () => {
    respondWith(200, {
      domain: "demo.example",
      organic_traffic: 1844,
      organic_keywords: 541,
      top_keywords: [{ keyword: "ferritin levels", position: 2, volume: 9900, traffic: 610, cpc: 0.4, url: "https://demo.example/f" }],
      top_pages: [{ url: "https://demo.example/f", traffic: 402.9, keywords: 58 }],
      cached: true,
      cached_at: "2026-08-27T10:00:00Z",
    });
    await run("domain");
    expect(calls[0]?.body).toEqual({});
    expect(output).toBe(
      [
        "domain=demo.example organic_traffic=1844 organic_keywords=541",
        "top keywords",
        "keyword          position  volume  traffic  cpc  url",
        "ferritin levels         2    9900      610  0.4  https://demo.example/f",
        "top pages",
        "url                     traffic  keywords",
        "https://demo.example/f    402.9        58",
        "cached 2026-08-27T10:00:00Z — nothing spent",
        "",
      ].join("\n"),
    );
  });
});

describe("mentions", () => {
  it("sends brand, competitors and dry_run", async () => {
    respondWith(200, { estimate: { cost: 1.08, requests: 8 } });
    await run("mentions", "--brand", "Demo", "--competitor", "Rival One", "--competitor", "Rival Two", "--dry-run");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/llm_mentions");
    expect(calls[0]?.body).toEqual({ brand: "Demo", competitors: ["Rival One", "Rival Two"], dry_run: true });
    expect(output).toBe("estimate $1.08 (requests=8) — dry run, nothing spent\n");
  });

  it("renders per-platform mentions, share of voice, prompts and cost", async () => {
    respondWith(200, {
      brand: "Demo",
      platforms: {
        chat_gpt: { mentions: 42, ai_search_volume: 15300, top_pages: [{ url: "https://demo.example/f", mentions: 17 }], sample_prompts: [{ question: "normal ferritin", ai_search_volume: 8100, cites_own: true }] },
        google: { mentions: 30, ai_search_volume: 9000, top_pages: [], sample_prompts: [] },
      },
      share_of_voice: { Demo: 100 },
      cached: false,
      cost: 0.85,
    });
    await run("mentions");
    expect(calls[0]?.body).toEqual({});
    expect(output).toContain("platform  mentions  ai_search_volume  top_page");
    expect(output).toContain("chat_gpt        42             15300  https://demo.example/f");
    expect(output).toContain("share of voice Demo=100");
    expect(output).toContain("chat_gpt prompts");
    expect(output).toContain("normal ferritin              8100  yes");
    expect(output.trimEnd().endsWith("cost $0.85")).toBe(true);
  });
});

describe("audit", () => {
  it("run enqueues with lighthouse flags and prints the run id", async () => {
    respondWith(202, { status: "enqueued", run_id: 7, estimate: { cost: 0.05 } });
    await run("audit", "run", "--lighthouse", "--pages", "10", "--max-pages", "200");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/site_audits");
    expect(calls[0]?.body).toEqual({ lighthouse: true, pages: 10, max_pages: 200 });
    expect(output).toBe("site audit run 7 enqueued — seo audit show 7\n");
  });

  it("run --dry-run prints the estimate", async () => {
    respondWith(200, { estimate: { cost: 0, lighthouse_pages: 0 } });
    await run("audit", "--dry-run");
    expect(calls[0]?.body).toEqual({ dry_run: true });
    expect(output).toBe("estimate $0.0 (lighthouse_pages=0) — dry run, nothing spent\n");
  });

  it("show renders summary, issues and lighthouse", async () => {
    respondWith(200, {
      site_audit_run: {
        id: 7,
        status: "completed",
        pages_count: 3,
        issue_counts: { critical: 1, warning: 0, info: 0 },
        cost: 0.01,
        issues: [{ severity: "critical", rule: "sitemap_non_200", url: "https://demo.example/gone", detail: "HTTP 404", how_to_fix: "fix" }],
        summary: { key_pages: { "/": 200, "/robots.txt": 200 }, retired: {}, lighthouse: { "https://demo.example/": { performance: 0.41, seo: 0.92, accessibility: 0.88, best_practices: 0.96 } } },
      },
    });
    await run("audit", "show", "7");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/site_audits/7");
    expect(output).toContain('issues  {"critical":1,"warning":0,"info":0}');
    expect(output).toContain("key pages /=200 /robots.txt=200");
    expect(output).toContain("critical  sitemap_non_200  https://demo.example/gone  HTTP 404");
    expect(output).toContain("https://demo.example/         0.41  0.92           0.88            0.96");
  });

  it("list prints runs", async () => {
    respondWith(200, { site_audit_runs: [{ id: 7, status: "completed", pages_count: 3, issue_counts: { critical: 1, warning: 2, info: 5 }, cost: 0, created_at: "2026-08-27" }] });
    await run("audit", "list", "--limit", "5");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/site_audits?limit=5");
    expect(output.split("\n")[0]).toBe("id  status     pages  critical  warning  info  cost  created_at");
  });
});

describe("backlinks --history --rows", () => {
  it("sends history and rows and renders both tables", async () => {
    respondWith(200, {
      domain: "demo.example",
      snapshot: { domain: "demo.example", measured_on: "2026-08-27", referring_domains: 11, backlinks: 84, rank: 27, spam_score: 3 },
      history: [{ month: "2026-07", backlinks: 86, referring_domains: 11, new: 2, lost: 1 }],
      referring_domains: [{ domain: "ref.test", rank: 300, backlinks: 12, dofollow: 10, first_seen: "2026-01-04" }],
      cost: 0.0796,
    });
    await run("backlinks", "--history", "--rows", "200");
    expect(calls[0]?.body).toEqual({ history: true, rows: 200 });
    expect(output).toContain("history\nmonth    backlinks  referring_domains  new  lost\n2026-07         86                 11    2     1");
    expect(output).toContain("referring domains\ndomain    rank  backlinks  dofollow  first_seen\nref.test   300         12        10  2026-01-04");
    expect(output.trimEnd().endsWith("cost $0.0796")).toBe(true);
  });

  it("estimate backlinks --history forwards dry_run", async () => {
    respondWith(200, { estimate: { cost: 0.0796 } });
    await run("estimate", "backlinks", "--history", "--rows", "200");
    expect(calls[0]?.body).toEqual({ history: true, rows: 200, dry_run: true });
  });
});

describe("keywords update", () => {
  it("patches path, track and status", async () => {
    respondWith(200, { keyword: { id: 3, keyword: "seo cli", set_name: "guarantee", status: "paused", track: "bofu", target_path: "/seo-cli", locked: false } });
    await run("keywords", "update", "3", "--path", "/seo-cli", "--track", "bofu", "--status", "paused");
    expect(calls[0]?.url).toBe("http://backend.test/api/v1/projects/demo/keywords/3");
    expect(calls[0]?.method).toBe("PATCH");
    expect(calls[0]?.body).toEqual({ target_path: "/seo-cli", track: "bofu", status: "paused" });
    expect(output).toBe("id  keyword  set        status  track  path      locked\n 3  seo cli  guarantee  paused  bofu   /seo-cli  no\n");
  });

  it("refuses an empty update", async () => {
    await expect(run("keywords", "update", "3")).rejects.toThrow(/--path, --track, --status/);
    expect(calls).toHaveLength(0);
  });
});
