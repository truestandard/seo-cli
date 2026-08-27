import type { Command } from "commander";
import { intOption, buildContext, emit } from "../context.js";
import { asObject, keyValueBlock } from "../output.js";

type CreateOptions = {
  name?: string;
  domain?: string;
  repoPath?: string;
  locationCode?: number;
  languageCode?: string;
};

export function registerProject(program: Command): void {
  const project = program.command("project").description("show or create a project");

  project
    .command("show [slug]", { isDefault: true })
    .description("show a project (defaults to the current one)")
    .action(async (slug: string | undefined, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.getProject(slug ?? context.requireProject());
      emit(context, data, () => keyValueBlock(projectFields(data)));
    });

  project
    .command("create <slug>")
    .description("create a project")
    .requiredOption("--name <name>", "display name")
    .requiredOption("--domain <domain>", "site domain, e.g. example.com")
    .option("--repo-path <path>", "local repo path")
    .option("--location-code <code>", "DataForSEO location code (default 2840)", intOption("--location-code"))
    .option("--language-code <code>", "language code (default en)")
    .action(async (slug: string, options: CreateOptions, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.createProject({
        slug,
        ...(options.name !== undefined && { name: options.name }),
        ...(options.domain !== undefined && { domain: options.domain }),
        ...(options.repoPath !== undefined && { repo_path: options.repoPath }),
        ...(options.locationCode !== undefined && { location_code: options.locationCode }),
        ...(options.languageCode !== undefined && { language_code: options.languageCode }),
      });
      emit(context, data, () => `created project ${slug}\n${keyValueBlock(projectFields(data))}`);
    });
}

function projectFields(data: unknown): Record<string, unknown> {
  return asObject(asObject(data)?.project) ?? asObject(data) ?? {};
}
