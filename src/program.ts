import { Command } from "commander";
import { registerAi } from "./commands/ai.js";
import { registerBacklinks } from "./commands/backlinks.js";
import { registerContext } from "./commands/context.js";
import { registerEstimate } from "./commands/estimate.js";
import { registerExperiments } from "./commands/experiments.js";
import { registerFloor } from "./commands/floor.js";
import { registerGsc } from "./commands/gsc.js";
import { registerKeywords } from "./commands/keywords.js";
import { registerLog } from "./commands/log.js";
import { registerLogin } from "./commands/login.js";
import { registerMcp } from "./commands/mcp.js";
import { registerProject } from "./commands/project.js";
import { registerProjects } from "./commands/projects.js";
import { registerPrompts } from "./commands/prompts.js";
import { registerRanks } from "./commands/ranks.js";
import { registerResearch } from "./commands/research.js";
import { registerScoreboard } from "./commands/scoreboard.js";
import { registerSerp } from "./commands/serp.js";
import { registerShip } from "./commands/ship.js";
import { registerSpend } from "./commands/spend.js";
import { registerTrack } from "./commands/track.js";
import { registerUse } from "./commands/use.js";
import { packageVersion } from "./version.js";

export function buildProgram(): Command {
  const program = new Command("seo")
    .description("CLI for the seo backend: keywords, ranks, AI visibility, floors, spend")
    .version(packageVersion())
    .option("--json", "print the raw API response as JSON")
    .option("--project <slug>", "project slug (overrides `seo use` and SEO_PROJECT)")
    .option("--dry-run", "for paid commands: print the cost estimate and spend nothing")
    .option("--base-url <url>", "backend URL (overrides config and SEO_BASE_URL)")
    .showHelpAfterError()
    .configureOutput({ writeErr: (text) => process.stderr.write(text) });

  registerLogin(program);
  registerProjects(program);
  registerUse(program);
  registerProject(program);
  registerContext(program);
  registerKeywords(program);
  registerResearch(program);
  registerSerp(program);
  registerTrack(program);
  registerRanks(program);
  registerPrompts(program);
  registerAi(program);
  registerFloor(program);
  registerBacklinks(program);
  registerGsc(program);
  registerShip(program);
  registerExperiments(program);
  registerLog(program);
  registerScoreboard(program);
  registerSpend(program);
  registerEstimate(program, buildProgram);
  registerMcp(program);
  return program;
}
