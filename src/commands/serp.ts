import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, summaryLine, table } from "../output.js";

export function registerSerp(program: Command): void {
  program
    .command("serp <keyword>")
    .description("live SERP for one keyword (paid: DataForSEO SERP live)")
    .option("--depth <n>", "results depth (default 100)", intOption("--depth"))
    .action(async (keyword: string, options: { depth?: number }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.serp(context.requireProject(), {
        keyword,
        ...(options.depth !== undefined && { depth: options.depth }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => {
        if (isDryRunEstimate(data)) return estimateLine(data);
        const result = asObject(asObject(data)?.result) ?? asObject(data) ?? {};
        const lines = [
          summaryLine({
            keyword,
            position: result.position,
            rank_absolute: result.rank_absolute,
            url: result.url,
            path_match: result.path_match,
            features: result.features,
          }),
        ];
        const topDomains = asRows(result.top_domains);
        if (topDomains.length > 0) lines.push(table(topDomains, ["rank", "domain", "url", "title"]));
        return lines.join("\n");
      });
    });
}
