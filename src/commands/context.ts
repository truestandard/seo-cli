import type { Command } from "commander";
import { readFileSync } from "node:fs";
import type { JsonObject } from "../client.js";
import { buildContext, emit } from "../context.js";
import { asObject, asRows, formatCell, keyValueBlock, table } from "../output.js";

export function registerContext(program: Command): void {
  const context = program.command("context").description("project memory: sections, competitors, key pages, research log");

  context
    .command("get", { isDefault: true })
    .description("print the project context")
    .action(async (_options: unknown, command: Command) => {
      const run = buildContext(command);
      const data = await run.client.getContext(run.requireProject());
      emit(run, data, () => renderContext(data));
    });

  context
    .command("set <key> [text...]")
    .description("set a context section (reads stdin when text is omitted)")
    .option("--file <path>", "read the section content from a file")
    .action(async (key: string, text: string[], options: { file?: string }, command: Command) => {
      const run = buildContext(command);
      const content = options.file ? readFileSync(options.file, "utf8") : text.length > 0 ? text.join(" ") : await readStdin();
      const data = await run.client.updateContext(run.requireProject(), { sections: { [key]: content } });
      emit(run, data, () => `set section ${key} (${content.length} chars)`);
    });

  context
    .command("add-competitor <domain>")
    .description("add a competitor domain")
    .option("--name <name>", "competitor name")
    .option("--notes <notes>", "notes")
    .action(async (domain: string, options: { name?: string; notes?: string }, command: Command) => {
      const run = buildContext(command);
      const data = await run.client.updateContext(run.requireProject(), {
        add_competitors: [{ domain, ...(options.name && { name: options.name }), ...(options.notes && { notes: options.notes }) }],
      });
      emit(run, data, () => `added competitor ${domain}`);
    });

  context
    .command("add-page <path>")
    .description("add a key page")
    .option("--role <role>", "page role, e.g. landing, pricing, blog")
    .option("--topic <topic>", "topic the page targets")
    .action(async (path: string, options: { role?: string; topic?: string }, command: Command) => {
      const run = buildContext(command);
      const data = await run.client.updateContext(run.requireProject(), {
        add_key_pages: [{ path, ...(options.role && { role: options.role }), ...(options.topic && { topic: options.topic }) }],
      });
      emit(run, data, () => `added key page ${path}`);
    });

  context
    .command("log <kind> <summary...>")
    .description("append a research log entry")
    .option("--inputs <json>", "JSON object describing the inputs")
    .action(async (kind: string, summary: string[], options: { inputs?: string }, command: Command) => {
      const run = buildContext(command);
      const inputs = options.inputs ? parseJsonObject(options.inputs) : undefined;
      const data = await run.client.updateContext(run.requireProject(), {
        research_log: { kind, summary: summary.join(" "), ...(inputs && { inputs }) },
      });
      emit(run, data, () => `logged ${kind}`);
    });
}

const CONTEXT_COLLECTIONS = ["sections", "competitors", "key_pages", "research_log"];

function renderContext(data: unknown): string {
  const root = asObject(asObject(data)?.context) ?? asObject(data) ?? {};
  const parts: string[] = [];
  const sections = root.sections;
  if (Array.isArray(sections)) {
    for (const section of asRows(sections)) parts.push(`## ${formatCell(section.key)}\n${formatCell(section.content)}`);
  } else if (asObject(sections)) {
    for (const [key, content] of Object.entries(asObject(sections) ?? {})) {
      parts.push(`## ${key}\n${typeof content === "string" ? content : formatCell(asObject(content)?.content ?? content)}`);
    }
  }
  const competitors = asRows(root.competitors);
  if (competitors.length > 0) parts.push(`## competitors\n${table(competitors, ["domain", "name", "notes"])}`);
  const keyPages = asRows(root.key_pages);
  if (keyPages.length > 0) parts.push(`## key pages\n${table(keyPages, ["path", "role", "topic"])}`);
  const researchLog = asRows(root.research_log);
  if (researchLog.length > 0) parts.push(`## research log\n${table(researchLog, ["created_at", "kind", "summary"])}`);
  if (parts.length === 0) {
    const remaining = Object.fromEntries(Object.entries(root).filter(([key]) => !CONTEXT_COLLECTIONS.includes(key)));
    return Object.keys(remaining).length > 0 ? keyValueBlock(remaining) : "context is empty";
  }
  return parts.join("\n\n");
}

function parseJsonObject(text: string): JsonObject {
  const parsed: unknown = JSON.parse(text);
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) throw new Error("--inputs must be a JSON object");
  return parsed as JsonObject;
}

async function readStdin(): Promise<string> {
  let text = "";
  for await (const chunk of process.stdin) text += chunk;
  return text.trimEnd();
}
