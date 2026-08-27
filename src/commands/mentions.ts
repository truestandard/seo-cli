import type { Command } from "commander";
import { buildContext, collectOption, emit } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, money, summaryLine, table } from "../output.js";

type Options = { brand?: string; competitor?: string[]; force?: boolean };

export function registerMentions(program: Command): void {
  program
    .command("mentions")
    .description("brand mentions in ChatGPT and Google AI answers with share of voice (paid: DataForSEO LLM Mentions, about $0.85; cached 24h)")
    .option("--brand <name>", "brand name (default: the project name)")
    .option("--competitor <name>", "competitor brand, repeatable (+$0.23 each)", collectOption, [])
    .option("--force", "bypass the 24h cache")
    .action(async (options: Options, command: Command) => {
      const context = buildContext(command);
      const competitors = options.competitor ?? [];
      const data = await context.client.llmMentions(context.requireProject(), {
        ...(options.brand !== undefined && { brand: options.brand }),
        ...(competitors.length > 0 && { competitors }),
        ...(options.force && { force: true }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => (isDryRunEstimate(data) ? estimateLine(data) : renderMentions(data)));
    });
}

export function renderMentions(data: unknown): string {
  const root = asObject(data) ?? {};
  const platforms = asObject(root.platforms) ?? {};
  const rows = Object.entries(platforms).map(([platform, report]) => {
    const details = asObject(report) ?? {};
    const topPage = asRows(details.top_pages)[0];
    return {
      platform,
      mentions: details.mentions,
      ai_search_volume: details.ai_search_volume,
      top_page: topPage?.url,
    };
  });
  const lines = [summaryLine({ brand: root.brand, fetched_at: root.fetched_at })];
  if (rows.length > 0) lines.push(table(rows, ["platform", "mentions", "ai_search_volume", "top_page"]));
  const share = asObject(root.share_of_voice);
  if (share) lines.push(`share of voice ${summaryLine(share)}`);
  for (const [platform, report] of Object.entries(platforms)) {
    const prompts = asRows(asObject(report)?.sample_prompts).slice(0, 5);
    if (prompts.length > 0) lines.push(`${platform} prompts`, table(prompts, ["question", "ai_search_volume", "cites_own"]));
  }
  lines.push(root.cached ? `cached ${String(root.cached_at ?? "")} — nothing spent`.trim() : `cost ${money(root.cost)}`);
  return lines.join("\n");
}
