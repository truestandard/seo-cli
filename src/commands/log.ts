import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asRows, table } from "../output.js";

export function registerLog(program: Command): void {
  program
    .command("log")
    .description("research log entries")
    .option("--kind <kind>", "filter by kind")
    .option("--days <n>", "look back this many days (default 30)", intOption("--days"))
    .action(async (options: { kind?: string; days?: number }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.researchLog(context.requireProject(), { kind: options.kind, days: options.days });
      emit(context, data, () => {
        const rows = asRows(data, "entries", "research_log");
        return rows.length > 0 ? table(rows, ["created_at", "kind", "summary", "cost", "actor"]) : "no log entries";
      });
    });
}
