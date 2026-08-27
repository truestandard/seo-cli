import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asRows, table } from "../output.js";

export function registerProjects(program: Command): void {
  program
    .command("projects")
    .description("list projects")
    .action(async (_options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.listProjects();
      emit(context, data, () => {
        const rows = asRows(data, "projects");
        if (rows.length === 0) return "no projects";
        return table(rows, ["slug", "name", "domain", "location_code", "language_code"]);
      });
    });
}
