import type { Command } from "commander";
import { buildContext, emit } from "../context.js";
import { asObject, summaryLine } from "../output.js";

type ShipOptions = { keyword?: string; track?: string; note?: string };

export function registerShip(program: Command): void {
  program
    .command("ship <url>")
    .description("record a shipped page and verify it responds with a 200 and an h1")
    .option("--keyword <keyword>", "keyword the page targets")
    .option("--track <track>", "track label")
    .option("--note <note>", "free-text note")
    .action(async (url: string, options: ShipOptions, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.ship(context.requireProject(), {
        url,
        ...(options.keyword !== undefined && { keyword: options.keyword }),
        ...(options.track !== undefined && { track: options.track }),
        ...(options.note !== undefined && { note: options.note }),
      });
      emit(context, data, () => {
        const event = asObject(asObject(data)?.ship_event) ?? asObject(asObject(data)?.ship) ?? asObject(data) ?? {};
        return summaryLine({
          url: event.url ?? url,
          verified: event.verified,
          http_status: event.http_status,
          h1: event.h1,
          keyword: event.keyword ?? options.keyword,
        });
      });
    });
}
