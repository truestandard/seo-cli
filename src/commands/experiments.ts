import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asObject, asRows, keyValueBlock, table } from "../output.js";

type AddOptions = { page?: string; hypothesis?: string; keyword?: string; shippedOn?: string };

const EXPERIMENT_COLUMNS = ["id", "shipped_on", "page", "change", "hypothesis", "outcome", "outcome_on"];

export function registerExperiments(program: Command): void {
  const experiments = program.command("experiments").description("on-page experiments and their outcomes");

  experiments
    .command("list", { isDefault: true })
    .description("list experiments")
    .action(async (_options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.listExperiments(context.requireProject());
      emit(context, data, () => {
        const rows = asRows(data, "experiments");
        return rows.length > 0 ? table(rows, EXPERIMENT_COLUMNS) : "no experiments";
      });
    });

  experiments
    .command("add <change...>")
    .description("record an experiment (what changed)")
    .option("--page <path>", "page path the change shipped on")
    .option("--hypothesis <text>", "why it should move the needle")
    .option("--keyword <keyword>", "keyword it targets")
    .option("--shipped-on <date>", "YYYY-MM-DD (default today)")
    .action(async (change: string[], options: AddOptions, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.createExperiment(context.requireProject(), {
        change: change.join(" "),
        ...(options.page !== undefined && { page: options.page }),
        ...(options.hypothesis !== undefined && { hypothesis: options.hypothesis }),
        ...(options.keyword !== undefined && { keyword: options.keyword }),
        ...(options.shippedOn !== undefined && { shipped_on: options.shippedOn }),
      });
      emit(context, data, () => keyValueBlock(experimentFields(data)));
    });

  experiments
    .command("outcome <id> <outcome...>")
    .description("record the outcome of an experiment")
    .action(async (id: string, outcome: string[], _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.updateExperiment(context.requireProject(), id, { outcome: outcome.join(" ") });
      emit(context, data, () => keyValueBlock(experimentFields(data)));
    });
}

function experimentFields(data: unknown): Record<string, unknown> {
  const experiment = asObject(asObject(data)?.experiment) ?? asObject(data) ?? {};
  return Object.fromEntries(EXPERIMENT_COLUMNS.filter((key) => experiment[key] !== undefined).map((key) => [key, experiment[key]]));
}
