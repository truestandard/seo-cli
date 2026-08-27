import type { Command } from "commander";
import type { KeywordInput } from "../client.js";
import { buildContext, emit, intOption } from "../context.js";
import { asRows, table } from "../output.js";

type AddOptions = { track?: string; path?: string; set?: string; locked?: boolean; volume?: number; kd?: number };

const KEYWORD_HEADERS = { set_name: "set", target_path: "path" };

export function registerKeywords(program: Command): void {
  const keywords = program.command("keywords").description("list, add, or remove tracked keywords");

  keywords
    .command("list", { isDefault: true })
    .description("list keywords")
    .option("--set <name>", "filter by set name")
    .action(async (options: { set?: string }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.listKeywords(context.requireProject(), { set: options.set });
      emit(context, data, () => {
        const rows = asRows(data, "keywords");
        if (rows.length === 0) return "no keywords";
        return table(rows, ["id", "keyword", "set_name", "track", "target_path", "volume", "kd", "locked"], KEYWORD_HEADERS);
      });
    });

  keywords
    .command("add <keywords...>")
    .description("add keywords")
    .option("--track <track>", "track label, e.g. bofu, brand")
    .option("--path <path>", "target path on the site")
    .option("--set <name>", "set name (default guarantee)")
    .option("--locked", "lock the keywords as a measurement contract (guarantee sets)")
    .option("--volume <n>", "monthly volume if already known", intOption("--volume"))
    .option("--kd <n>", "keyword difficulty if already known", intOption("--kd"))
    .action(async (texts: string[], options: AddOptions, command: Command) => {
      const context = buildContext(command);
      const payload: KeywordInput[] = texts.map((keyword) => ({
        keyword,
        ...(options.track !== undefined && { track: options.track }),
        ...(options.path !== undefined && { target_path: options.path }),
        ...(options.set !== undefined && { set_name: options.set }),
        ...(options.locked !== undefined && { locked: options.locked }),
        ...(options.volume !== undefined && { volume: options.volume }),
        ...(options.kd !== undefined && { kd: options.kd }),
      }));
      const data = await context.client.addKeywords(context.requireProject(), payload);
      emit(context, data, () => {
        const rows = asRows(data, "keywords");
        return rows.length > 0
          ? table(rows, ["id", "keyword", "set_name", "track", "target_path", "locked"], KEYWORD_HEADERS)
          : `added ${texts.length} keyword${texts.length === 1 ? "" : "s"}`;
      });
    });

  keywords
    .command("remove <ids...>")
    .description("remove keywords by id")
    .action(async (ids: string[], _options: unknown, command: Command) => {
      const context = buildContext(command);
      const slug = context.requireProject();
      const responses: unknown[] = [];
      for (const id of ids) responses.push(await context.client.deleteKeyword(slug, id));
      emit(context, { removed: ids, responses }, () => `removed ${ids.join(", ")}`);
    });
}
