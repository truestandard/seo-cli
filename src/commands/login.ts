import type { Command } from "commander";
import { createInterface } from "node:readline/promises";
import { SeoClient } from "../client.js";
import { resolveConfig, stripTrailingSlash, updateStoredConfig } from "../config.js";
import { buildContext, emit, globalOptions } from "../context.js";
import { asObject, summaryLine } from "../output.js";

const CTRL_C = "\u0003";
const DELETE = "\u007f";

export function registerLogin(program: Command): void {
  program
    .command("login")
    .description("save the backend URL and API token after verifying them with GET /api/v1/whoami")
    .option("--token <token>", "API token (skips the prompt)")
    .action(async (options: { token?: string }, command: Command) => {
      const context = buildContext(command);
      const explicitBaseUrl = globalOptions(command).baseUrl;
      const current = resolveConfig();
      const baseUrl = stripTrailingSlash(
        explicitBaseUrl ?? (process.stdin.isTTY ? (await ask(`Backend URL [${current.baseUrl}]: `)) || current.baseUrl : current.baseUrl),
      );
      const token = options.token ?? (await askHidden("API token (seo_…): "));
      if (!token) throw new Error("a token is required; create one in the backend and paste it here");
      const client = new SeoClient({ baseUrl, token });
      const identity = await client.whoami();
      updateStoredConfig({ baseUrl, token });
      emit(context, { baseUrl, whoami: identity }, () => {
        const details = summaryLine(asObject(identity) ?? {});
        return `logged in to ${baseUrl}${details ? ` (${details})` : ""}`;
      });
    });
}

async function ask(prompt: string): Promise<string> {
  const readline = createInterface({ input: process.stdin, output: process.stderr });
  try {
    return (await readline.question(prompt)).trim();
  } finally {
    readline.close();
  }
}

function askHidden(prompt: string): Promise<string> {
  const input = process.stdin;
  if (!input.isTTY) return readAllStdin();
  process.stderr.write(prompt);
  return new Promise((resolve, reject) => {
    let buffer = "";
    input.setRawMode(true);
    input.resume();
    input.setEncoding("utf8");
    const finish = (error?: Error) => {
      input.setRawMode(false);
      input.pause();
      input.off("data", onData);
      process.stderr.write("\n");
      if (error) reject(error);
      else resolve(buffer.trim());
    };
    const onData = (chunk: string) => {
      for (const char of chunk) {
        if (char === CTRL_C) return finish(new Error("cancelled"));
        if (char === "\r" || char === "\n") return finish();
        if (char === DELETE || char === "\b") buffer = buffer.slice(0, -1);
        else buffer += char;
      }
    };
    input.on("data", onData);
  });
}

async function readAllStdin(): Promise<string> {
  let text = "";
  for await (const chunk of process.stdin) text += chunk;
  return text.trim();
}
