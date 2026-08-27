import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asRows, estimateLine, isDryRunEstimate, money, pickObject, table } from "../output.js";

type ResearchOptions = { limit?: number; maxKd?: number; keywords?: boolean };

export function registerResearch(program: Command): void {
  program
    .command("research <seeds...>")
    .description("keyword research from seed terms (paid: DataForSEO Labs)")
    .option("--limit <n>", "max suggestions per seed (default 40)", intOption("--limit"))
    .option("--max-kd <n>", "max keyword difficulty (default 30)", intOption("--max-kd"))
    .option("--keywords", "treat the arguments as exact keywords for an overview instead of seeds")
    .action(async (seeds: string[], options: ResearchOptions, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.research(context.requireProject(), {
        ...(options.keywords ? { keywords: seeds } : { seeds }),
        ...(options.limit !== undefined && { limit: options.limit }),
        ...(options.maxKd !== undefined && { max_kd: options.maxKd }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => {
        if (isDryRunEstimate(data)) return estimateLine(data);
        const rows = asRows(data, "keywords", "suggestions");
        if (rows.length === 0) return "no keywords returned";
        const lines = [table(rows, ["keyword", "volume", "kd", "cpc", "intent", "yoy"])];
        const cost = pickObject(data, "spend")?.cost ?? (data as { cost?: unknown } | null)?.cost;
        if (cost !== undefined) lines.push(`cost ${money(cost)}`);
        return lines.join("\n");
      });
    });
}
