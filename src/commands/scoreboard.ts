import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asObject, keyValueBlock, money, summaryLine } from "../output.js";

export function registerScoreboard(program: Command): void {
  program
    .command("scoreboard")
    .description("floors, rank summary, AI summary, ships, and 30-day spend for the project")
    .action(async (_options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.scoreboard(context.requireProject());
      emit(context, data, () => {
        const board = asObject(asObject(data)?.scoreboard) ?? asObject(data) ?? {};
        const lines: string[] = [];
        const floors = asObject(board.floors);
        if (floors && Object.keys(floors).length > 0) lines.push(`floors  ${summaryLine(floors)}`);
        const ranks = asObject(board.ranks);
        if (ranks) lines.push(`ranks   ${summaryLine(ranks)}`);
        const ai = asObject(board.ai);
        if (ai) lines.push(`ai      ${summaryLine(ai)}`);
        lines.push(
          keyValueBlock({
            ships: board.ships,
            spend_30d: board.spend_30d !== undefined ? money(board.spend_30d) : undefined,
            last_rank_run_on: board.last_rank_run_on,
            last_ai_scan_on: board.last_ai_scan_on,
          }),
        );
        return lines.join("\n");
      });
    });
}
