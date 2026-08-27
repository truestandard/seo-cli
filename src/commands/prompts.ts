import type { Command } from "commander";
import { readFileSync } from "node:fs";
import type { PromptInput } from "../client.js";
import { buildContext, emit } from "../context.js";
import { asRows, table } from "../output.js";

type AddOptions = { set?: string; locked?: boolean; file?: string };

export function registerPrompts(program: Command): void {
  const prompts = program.command("prompts").description("AI visibility prompt sets");

  prompts
    .command("list", { isDefault: true })
    .description("list prompts")
    .option("--set <name>", "filter by set name")
    .action(async (options: { set?: string }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.listAiPrompts(context.requireProject(), { set: options.set });
      emit(context, data, () => {
        const rows = asRows(data, "prompts", "ai_prompts");
        if (rows.length === 0) return "no prompts";
        return table(rows, ["id", "set_name", "locked", "text"], { set_name: "set" });
      });
    });

  prompts
    .command("add [texts...]")
    .description("add prompts (one per argument, or one per line with --file)")
    .option("--set <name>", "set name (default guarantee)")
    .option("--locked", "lock the prompts as a measurement contract (guarantee sets)")
    .option("--file <path>", "read prompts from a file, one per line")
    .action(async (texts: string[], options: AddOptions, command: Command) => {
      const context = buildContext(command);
      const fromFile = options.file
        ? readFileSync(options.file, "utf8")
            .split("\n")
            .map((line) => line.trim())
            .filter((line) => line !== "")
        : [];
      const all = [...texts, ...fromFile];
      if (all.length === 0) throw new Error("no prompts given: pass them as arguments or with --file");
      const payload: PromptInput[] = all.map((text) => ({
        text,
        ...(options.set !== undefined && { set_name: options.set }),
        ...(options.locked !== undefined && { locked: options.locked }),
      }));
      const data = await context.client.addAiPrompts(context.requireProject(), payload);
      emit(context, data, () => {
        const rows = asRows(data, "prompts", "ai_prompts");
        return rows.length > 0
          ? table(rows, ["id", "set_name", "locked", "text"], { set_name: "set" })
          : `added ${all.length} prompt${all.length === 1 ? "" : "s"}`;
      });
    });
}
