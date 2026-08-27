import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, money, summaryLine, table } from "../output.js";

type Options = { limit?: number; force?: boolean };

export function registerDomain(program: Command): void {
  program
    .command("domain [domain]")
    .description("domain overview: organic traffic, keyword count, top keywords and pages (paid: DataForSEO Labs; cached 12h)")
    .option("--limit <n>", "top keywords to fetch (default 100)", intOption("--limit"))
    .option("--force", "bypass the 12h cache")
    .action(async (domain: string | undefined, options: Options, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.domainOverview(context.requireProject(), {
        ...(domain !== undefined && { domain }),
        ...(options.limit !== undefined && { limit: options.limit }),
        ...(options.force && { force: true }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => (isDryRunEstimate(data) ? estimateLine(data) : renderOverview(data)));
    });
}

export function renderOverview(data: unknown): string {
  const root = asObject(data) ?? {};
  const lines = [
    summaryLine({
      domain: root.domain,
      organic_traffic: root.organic_traffic,
      organic_keywords: root.organic_keywords,
      fetched_at: root.fetched_at,
    }),
  ];
  const keywords = asRows(root.top_keywords);
  if (keywords.length > 0) lines.push("top keywords", table(keywords, ["keyword", "position", "volume", "traffic", "cpc", "url"]));
  const pages = asRows(root.top_pages);
  if (pages.length > 0) lines.push("top pages", table(pages, ["url", "traffic", "keywords"]));
  lines.push(root.cached ? `cached ${String(root.cached_at ?? "")} — nothing spent`.trim() : `cost ${money(root.cost)}`);
  return lines.join("\n");
}
