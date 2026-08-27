import type { Command } from "commander";
import { since } from "../client.js";
import { buildContext, emit, intOption } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, keyValueBlock, summaryLine, table } from "../output.js";

export function registerAi(program: Command): void {
  const ai = program.command("ai").description("AI visibility scans");

  ai
    .command("run", { isDefault: true })
    .description("run an AI visibility scan (paid: OpenRouter)")
    .option("--set <name>", "prompt set (default guarantee)")
    .option("--runs <n>", "runs per prompt x model cell (default 1)", intOption("--runs"))
    .action(async (options: { set?: string; runs?: number }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.createAiScan(context.requireProject(), {
        ...(options.set !== undefined && { set_name: options.set }),
        ...(options.runs !== undefined && { runs_per_cell: options.runs }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => (isDryRunEstimate(data) ? estimateLine(data) : keyValueBlock(scanFields(data))));
    });

  ai
    .command("results")
    .description("AI visibility report and deltas since a date")
    .option("--since <window>", "7d, 30d, or YYYY-MM-DD", "30d")
    .option("--set <name>", "prompt set", "guarantee")
    .action(async (options: { since: string; set: string }, command: Command) => {
      const context = buildContext(command);
      const window = since(options.since);
      const data = await context.client.aiVisibility(context.requireProject(), { since: window.query, set: options.set });
      emit(context, data, () => renderVisibility(data));
    });

  ai
    .command("status <id>")
    .description("show an AI scan")
    .action(async (id: string, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.getAiScan(context.requireProject(), id);
      emit(context, data, () => keyValueBlock(scanFields(data)));
    });
}

function scanFields(data: unknown): Record<string, unknown> {
  const scan = asObject(asObject(data)?.ai_scan) ?? asObject(asObject(data)?.scan) ?? asObject(data) ?? {};
  return {
    id: scan.id,
    status: scan.status,
    run_on: scan.run_on,
    set_name: scan.set_name,
    models: scan.models,
    runs_per_cell: scan.runs_per_cell,
    cost: scan.cost,
    summary: scan.summary,
  };
}

function renderVisibility(data: unknown): string {
  const root = asObject(data) ?? {};
  const report = asObject(root.report) ?? asObject(root.summary) ?? root;
  const lines = [
    summaryLine({
      cells: report.cells,
      ok: report.ok,
      errors: report.errors,
      named: report.named,
      cited: report.cited,
      queries_hit: report.queries_hit,
      queries_always: report.queries_always,
    }),
  ];
  const perEngine = asObject(report.per_engine);
  if (perEngine) {
    const rows = Object.entries(perEngine).map(([engine, stats]) => ({ engine, ...(asObject(stats) ?? {}) }));
    lines.push(table(rows, ["engine", "cited", "with_urls", "blank"]));
  }
  const domains = asRows(report.domains_top);
  if (domains.length > 0) lines.push(table(domains));
  const rivals = asObject(report.rival_share);
  if (rivals) lines.push(`rivals ${summaryLine(rivals)}`);
  const deltas = asRows(root.deltas);
  if (deltas.length > 0) lines.push(table(deltas));
  return lines.filter((line) => line !== "").join("\n");
}
