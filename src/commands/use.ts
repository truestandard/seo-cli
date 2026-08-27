import type { Command } from "commander";
import { updateStoredConfig } from "../config.js";
import { buildContext, emit } from "../context.js";

export function registerUse(program: Command): void {
  program
    .command("use <slug>")
    .description("set the default project")
    .action(async (slug: string, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const saved = updateStoredConfig({ project: slug });
      emit(context, { project: saved.project }, () => `project set to ${slug}`);
    });
}
