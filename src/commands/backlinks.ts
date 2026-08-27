import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, keyValueBlock, money, table } from "../output.js";

type Options = { history?: boolean; rows?: number; months?: number };

export function registerBacklinks(program: Command): void {
  program
    .command("backlinks [domain]")
    .description("backlink snapshot for a domain, optionally with monthly history and top referring domains (paid: DataForSEO backlinks)")
    .option("--history", "add 12 months of backlinks / referring domains / new / lost")
    .option("--months <n>", "months of history (default 12)", intOption("--months"))
    .option("--rows <n>", "list the top N referring domains by backlinks", intOption("--rows"))
    .action(async (domain: string | undefined, options: Options, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.backlinks(context.requireProject(), {
        ...(domain !== undefined && { domain }),
        ...(options.history && { history: true }),
        ...(options.months !== undefined && { months: options.months }),
        ...(options.rows !== undefined && { rows: options.rows }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => (isDryRunEstimate(data) ? estimateLine(data) : renderBacklinks(data)));
    });
}

export function renderBacklinks(data: unknown): string {
  const root = asObject(data) ?? {};
  const snapshot = asObject(root.snapshot) ?? asObject(root.backlink_snapshot) ?? root;
  const lines = [
    keyValueBlock({
      domain: snapshot.domain,
      measured_on: snapshot.measured_on,
      referring_domains: snapshot.referring_domains,
      backlinks: snapshot.backlinks,
      rank: snapshot.rank,
      spam_score: snapshot.spam_score,
    }),
  ];
  const history = asRows(root.history);
  if (history.length > 0) lines.push("history", table(history, ["month", "backlinks", "referring_domains", "new", "lost"]));
  const referring = asRows(root.referring_domains);
  if (referring.length > 0) lines.push("referring domains", table(referring, ["domain", "rank", "backlinks", "dofollow", "first_seen"]));
  if (root.cost !== undefined) lines.push(`cost ${money(root.cost)}`);
  return lines.join("\n");
}
