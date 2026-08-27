import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asRows, estimateLine, isDryRunEstimate, table } from "../output.js";

export function registerFloor(program: Command): void {
  program
    .command("floor <keywords...>")
    .description("floor gate: weakest page-1 referring domains vs your own (paid: SERP live + backlinks)")
    .action(async (keywords: string[], _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.floor(context.requireProject(), {
        keywords,
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => {
        if (isDryRunEstimate(data)) return estimateLine(data);
        const rows = asRows(data, "probes", "floor_probes");
        if (rows.length === 0) return "no probes returned";
        return table(rows, ["keyword_text", "verdict", "own_referring_domains", "floor_referring_domains", "weakest_domain", "ratio"], {
          keyword_text: "keyword",
          own_referring_domains: "own_rds",
          floor_referring_domains: "floor_rds",
        });
      });
    });
}
