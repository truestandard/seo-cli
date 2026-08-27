import { describe, expect, it } from "vitest";
import { asRows, estimateLine, formatCell, keyValueBlock, money, summaryLine, table } from "../src/output.js";

describe("table", () => {
  it("aligns columns, right-aligns numbers, and never truncates keyword text", () => {
    const longKeyword = "best self hosted rank tracker for agencies with white label reporting and api access";
    const rendered = table([
      { keyword: "seo cli", pos: 3, prev: 12, url: "/blog/seo-cli" },
      { keyword: longKeyword, pos: 100, prev: null, url: undefined },
    ]);
    const lines = rendered.split("\n");
    expect(lines).toHaveLength(3);
    expect(lines[0]).toBe(`keyword${" ".repeat(longKeyword.length - "keyword".length)}  pos  prev  url`);
    expect(lines[1]).toBe(`seo cli${" ".repeat(longKeyword.length - "seo cli".length)}    3    12  /blog/seo-cli`);
    expect(lines[2]).toBe(`${longKeyword}  100     -  -`);
    expect(rendered).toContain(longKeyword);
  });

  it("uses the requested columns, headers, and order", () => {
    const rendered = table([{ set_name: "guarantee", keyword: "a", ignored: 1 }], ["keyword", "set_name"], { set_name: "set" });
    expect(rendered).toBe("keyword  set\na        guarantee");
  });

  it("returns an empty string for no rows", () => {
    expect(table([])).toBe("");
  });
});

describe("cells", () => {
  it("formats scalars, booleans, arrays, and objects", () => {
    expect(formatCell(null)).toBe("-");
    expect(formatCell("")).toBe("-");
    expect(formatCell(true)).toBe("yes");
    expect(formatCell(false)).toBe("no");
    expect(formatCell(3.14159)).toBe("3.1416");
    expect(formatCell(2.5)).toBe("2.5");
    expect(formatCell(["a", 1])).toBe("a,1");
    expect(formatCell({ a: 1 })).toBe('{"a":1}');
  });

  it("prints money to four decimals without trailing zeros", () => {
    expect(money(0.012)).toBe("$0.012");
    expect(money(1)).toBe("$1.0");
    expect(money("0.00060")).toBe("$0.0006");
    expect(money("n/a")).toBe("n/a");
  });
});

describe("summaries", () => {
  it("renders key=value summaries and key-value blocks", () => {
    expect(summaryLine({ top10: 3, floor_met: true, skipped: undefined })).toBe("top10=3 floor_met=yes");
    expect(keyValueBlock({ id: 1, status: "running" })).toBe("id      1\nstatus  running");
  });

  it("renders estimates from the dry_run envelope", () => {
    expect(estimateLine({ estimate: { cost: 0.024, keyword_count: 12, mode: "live" } })).toBe(
      "estimate $0.024 (keyword_count=12 mode=live) — dry run, nothing spent",
    );
  });

  it("unwraps bare arrays and keyed envelopes", () => {
    expect(asRows([{ a: 1 }, "junk"])).toEqual([{ a: 1 }]);
    expect(asRows({ keywords: [{ a: 1 }] }, "keywords")).toEqual([{ a: 1 }]);
    expect(asRows({ rows: [{ b: 2 }] })).toEqual([{ b: 2 }]);
    expect(asRows({ nothing: true })).toEqual([]);
    expect(asRows(null)).toEqual([]);
  });
});
