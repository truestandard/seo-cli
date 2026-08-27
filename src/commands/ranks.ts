import type { Command } from "commander";
import { since } from "../client.js";
import { buildContext, emit } from "../context.js";
import { asObject, asRows, formatCell, summaryLine, table } from "../output.js";

export function registerRanks(program: Command): void {
  program
    .command("ranks")
    .description("rank deltas since a date plus the set summary")
    .option("--since <window>", "7d, 30d, or YYYY-MM-DD", "7d")
    .option("--set <name>", "keyword set", "guarantee")
    .action(async (options: { since: string; set: string }, command: Command) => {
      const context = buildContext(command);
      const window = since(options.since);
      const data = await context.client.ranks(context.requireProject(), { since: window.query, set: options.set });
      emit(context, data, () => renderRanks(data));
    });
}

export function renderRanks(data: unknown): string {
  const rows = asRows(data, "rows").map((row) => ({
    keyword: row.keyword,
    pos: row.position,
    prev: row.previous,
    delta: signed(row.delta),
    band: row.band_change,
    url: row.url,
  }));
  const summary = asObject(asObject(data)?.summary);
  const lines: string[] = [];
  lines.push(rows.length > 0 ? table(rows) : "no ranked keywords in this window");
  if (summary) {
    lines.push(
      summaryLine({
        top10: summary.top10,
        top20: summary.top20,
        top100: summary.top100,
        unranked: summary.unranked,
        avg_position: summary.avg_position,
        floor_target: summary.floor_target,
        floor_met: summary.floor_met,
      }),
    );
  }
  return lines.join("\n");
}

function signed(value: unknown): string {
  if (typeof value !== "number") return formatCell(value);
  if (value > 0) return `+${value}`;
  return String(value);
}
