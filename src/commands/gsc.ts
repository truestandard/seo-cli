import type { Command } from "commander";
import { readFileSync } from "node:fs";
import { buildContext, emit } from "../context.js";
import { asObject, asRows, summaryLine, table } from "../output.js";

type ImportOptions = { dimension: string; start: string; end: string };

export function registerGsc(program: Command): void {
  const gsc = program.command("gsc").description("Google Search Console CSV import and opportunities");

  gsc
    .command("import <csv>")
    .description("import a GSC UI export (Top queries or Top pages)")
    .requiredOption("--dimension <query|page>", "which export this is")
    .requiredOption("--start <date>", "range start YYYY-MM-DD")
    .requiredOption("--end <date>", "range end YYYY-MM-DD")
    .action(async (csvPath: string, options: ImportOptions, command: Command) => {
      const context = buildContext(command);
      if (options.dimension !== "query" && options.dimension !== "page") throw new Error("--dimension must be query or page");
      const csv = readFileSync(csvPath, "utf8");
      const data = await context.client.gscImport(context.requireProject(), {
        dimension: options.dimension,
        range_start: options.start,
        range_end: options.end,
        csv,
      });
      emit(context, data, () => summaryLine({ imported: asObject(data)?.imported, dimension: options.dimension }));
    });

  gsc
    .command("striking")
    .description("striking-distance queries (position 8-20, by impressions)")
    .action(async (_options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.strikingDistance(context.requireProject());
      emit(context, data, () => {
        const rows = asRows(data, "queries", "striking_distance");
        if (rows.length === 0) return "no striking-distance queries";
        return table(rows, ["query", "position", "impressions", "clicks", "ctr"]);
      });
    });
}
