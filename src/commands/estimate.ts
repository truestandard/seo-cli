import type { Command } from "commander";
import { globalOptions } from "../context.js";

export const PAID_COMMANDS = ["research", "serp", "track", "ai", "floor", "backlinks", "domain", "mentions", "audit"];

export function registerEstimate(program: Command, buildProgram: () => Command): void {
  program
    .command("estimate <command> [args...]")
    .description(`re-run a paid command with dry_run and print only the estimate (${PAID_COMMANDS.join(", ")})`)
    .allowUnknownOption()
    .allowExcessArguments()
    .action(async (name: string, _args: string[], _options: unknown, command: Command) => {
      if (!PAID_COMMANDS.includes(name)) {
        throw new Error(`estimate only applies to paid commands: ${PAID_COMMANDS.join(", ")}`);
      }
      const globals = globalOptions(command);
      const forwarded = [
        ...(globals.json ? ["--json"] : []),
        ...(globals.project ? ["--project", globals.project] : []),
        ...(globals.baseUrl ? ["--base-url", globals.baseUrl] : []),
      ];
      const nested = buildProgram();
      nested.exitOverride();
      await nested.parseAsync([...command.args, ...forwarded, "--dry-run"], { from: "user" });
    });
}
