import type { Command } from "commander";
import { since } from "../client.js";
import { buildContext, emit } from "../context.js";
import { asObject, asRows, money, table } from "../output.js";

export function registerSpend(program: Command): void {
  program
    .command("spend")
    .description("spend across providers since a date")
    .option("--since <window>", "7d, 30d, or YYYY-MM-DD", "30d")
    .action(async (options: { since: string }, command: Command) => {
      const context = buildContext(command);
      const window = since(options.since);
      const data = await context.client.spend({ since: window.query });
      emit(context, data, () => {
        const root = asObject(data) ?? {};
        const lines: string[] = [];
        const byProvider = asRows(root.by_provider ?? root.providers);
        if (byProvider.length > 0) lines.push(table(byProvider));
        const entries = asRows(root.entries ?? root.spend_entries ?? root.rows);
        if (entries.length > 0) lines.push(table(entries, ["created_at", "project", "provider", "endpoint", "units", "cost"]));
        if (root.total !== undefined) lines.push(`total ${money(root.total)} since ${window.query}`);
        return lines.length > 0 ? lines.join("\n") : `no spend since ${window.query}`;
      });
    });
}
