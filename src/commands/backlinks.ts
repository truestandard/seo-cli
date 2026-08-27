import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asObject, estimateLine, isDryRunEstimate, keyValueBlock } from "../output.js";

export function registerBacklinks(program: Command): void {
  program
    .command("backlinks [domain]")
    .description("backlink snapshot for a domain (defaults to the project domain; paid: DataForSEO backlinks)")
    .action(async (domain: string | undefined, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.backlinks(context.requireProject(), {
        ...(domain !== undefined && { domain }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => {
        if (isDryRunEstimate(data)) return estimateLine(data);
        const snapshot = asObject(asObject(data)?.snapshot) ?? asObject(asObject(data)?.backlink_snapshot) ?? asObject(data) ?? {};
        return keyValueBlock({
          domain: snapshot.domain,
          measured_on: snapshot.measured_on,
          referring_domains: snapshot.referring_domains,
          backlinks: snapshot.backlinks,
          rank: snapshot.rank,
          spam_score: snapshot.spam_score,
        });
      });
    });
}
