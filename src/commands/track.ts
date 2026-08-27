import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asObject, estimateLine, isDryRunEstimate, keyValueBlock } from "../output.js";

type RunOptions = { live?: boolean; scheduled?: boolean; set?: string };

export function registerTrack(program: Command): void {
  const track = program.command("track").description("rank tracking runs");

  track
    .command("run", { isDefault: true })
    .description("start a rank run (paid: DataForSEO SERP)")
    .option("--live", "live SERP calls, completes now")
    .option("--scheduled", "standard queue, polled later (default)")
    .option("--set <name>", "keyword set (default guarantee)")
    .action(async (options: RunOptions, command: Command) => {
      const context = buildContext(command);
      if (options.live && options.scheduled) throw new Error("pass either --live or --scheduled, not both");
      const data = await context.client.createRankRun(context.requireProject(), {
        mode: options.live ? "live" : "scheduled",
        ...(options.set !== undefined && { set_name: options.set }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => (isDryRunEstimate(data) ? estimateLine(data) : keyValueBlock(rankRunFields(data))));
    });

  track
    .command("status <id>")
    .description("show a rank run")
    .action(async (id: string, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.getRankRun(context.requireProject(), id);
      emit(context, data, () => keyValueBlock(rankRunFields(data)));
    });
}

function rankRunFields(data: unknown): Record<string, unknown> {
  const run = asObject(asObject(data)?.rank_run) ?? asObject(data) ?? {};
  return {
    id: run.id,
    mode: run.mode,
    status: run.status,
    checked_on: run.checked_on,
    keyword_count: run.keyword_count,
    completed_count: run.completed_count,
    cost: run.cost,
    summary: run.summary,
  };
}
