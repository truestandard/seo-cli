import { InvalidArgumentError, type Command } from "commander";
import { printBlock, printJson } from "./output.js";
import { SeoClient } from "./client.js";
import { resolveConfig, type ResolvedConfig } from "./config.js";

export type GlobalOptions = {
  json?: boolean;
  project?: string;
  dryRun?: boolean;
  baseUrl?: string;
};

export type RunContext = {
  client: SeoClient;
  config: ResolvedConfig;
  json: boolean;
  dryRun: boolean;
  projectSlug: string | undefined;
  requireProject(): string;
};

export function globalOptions(command: Command): GlobalOptions {
  return command.optsWithGlobals<GlobalOptions>();
}

export function buildContext(command: Command): RunContext {
  const options = globalOptions(command);
  const config = resolveConfig();
  const baseUrl = options.baseUrl ?? config.baseUrl;
  const projectSlug = options.project ?? config.project;
  const client = new SeoClient({ baseUrl, token: config.token });
  return {
    client,
    config: { ...config, baseUrl, project: projectSlug },
    json: options.json === true,
    dryRun: options.dryRun === true,
    projectSlug,
    requireProject() {
      if (!projectSlug) {
        throw new Error("no project selected: run `seo use <slug>`, pass --project <slug>, or set SEO_PROJECT");
      }
      return projectSlug;
    },
  };
}

export function emit(context: RunContext, data: unknown, human: () => string): void {
  if (context.json) {
    printJson(data);
    return;
  }
  printBlock(human());
}

export function intOption(name: string): (value: string) => number {
  return (value) => {
    const parsed = Number(value);
    if (!Number.isInteger(parsed)) throw new InvalidArgumentError(`${name} must be an integer, got "${value}"`);
    return parsed;
  };
}

export function collectOption(value: string, previous: string[] = []): string[] {
  return [...previous, value];
}
