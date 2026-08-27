import type { Command } from "commander";
import { buildContext, emit, intOption } from "../context.js";
import { asObject, asRows, estimateLine, isDryRunEstimate, keyValueBlock, summaryLine, table } from "../output.js";

type RunOptions = { lighthouse?: boolean; maxPages?: number; pages?: number };

export function registerAudit(program: Command): void {
  const audit = program.command("audit").description("site audits: own crawler over the sitemap, key pages and retired paths");

  audit
    .command("run", { isDefault: true })
    .description("enqueue a site audit (crawl is free; --lighthouse is paid: DataForSEO $0.005/page)")
    .option("--lighthouse", "add Lighthouse mobile scores for the first pages")
    .option("--pages <n>", "pages to run Lighthouse on (default 20)", intOption("--pages"))
    .option("--max-pages <n>", "sitemap URLs to crawl (default 500)", intOption("--max-pages"))
    .action(async (options: RunOptions, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.createSiteAudit(context.requireProject(), {
        ...(options.lighthouse && { lighthouse: true }),
        ...(options.pages !== undefined && { pages: options.pages }),
        ...(options.maxPages !== undefined && { max_pages: options.maxPages }),
        ...(context.dryRun && { dry_run: true }),
      });
      emit(context, data, () => {
        if (isDryRunEstimate(data) && asObject(data)?.status === undefined) return estimateLine(data);
        const root = asObject(data) ?? {};
        return `site audit run ${String(root.run_id ?? "")} enqueued — seo audit show ${String(root.run_id ?? "")}`;
      });
    });

  audit
    .command("show <id>")
    .description("show a site audit run: summary, issues, pages")
    .action(async (id: string, _options: unknown, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.getSiteAudit(context.requireProject(), id);
      emit(context, data, () => renderAudit(data));
    });

  audit
    .command("list")
    .description("list site audit runs, newest first")
    .option("--limit <n>", "runs to list (default 20)", intOption("--limit"))
    .action(async (options: { limit?: number }, command: Command) => {
      const context = buildContext(command);
      const data = await context.client.listSiteAudits(context.requireProject(), { limit: options.limit });
      emit(context, data, () => {
        const rows = asRows(data, "site_audit_runs").map((run) => ({
          id: run.id,
          status: run.status,
          pages: run.pages_count,
          critical: asObject(run.issue_counts)?.critical,
          warning: asObject(run.issue_counts)?.warning,
          info: asObject(run.issue_counts)?.info,
          cost: run.cost,
          created_at: run.created_at,
        }));
        return rows.length > 0 ? table(rows) : "no site audit runs";
      });
    });
}

export function renderAudit(data: unknown): string {
  const run = asObject(asObject(data)?.site_audit_run) ?? asObject(data) ?? {};
  const summary = asObject(run.summary) ?? {};
  const lines = [
    keyValueBlock({
      id: run.id,
      status: run.status,
      pages: run.pages_count,
      issues: run.issue_counts,
      cost: run.cost,
      started_at: run.started_at,
      finished_at: run.finished_at,
      error: run.error,
    }),
  ];
  const keyPages = asObject(summary.key_pages);
  if (keyPages) lines.push(`key pages ${summaryLine(keyPages)}`);
  const retired = asObject(summary.retired);
  if (retired && Object.keys(retired).length > 0) lines.push(`retired ${summaryLine(retired)}`);
  const issues = asRows(run.issues);
  if (issues.length > 0) lines.push("issues", table(issues, ["severity", "rule", "url", "detail"]));
  const lighthouse = asObject(summary.lighthouse);
  if (lighthouse && Object.keys(lighthouse).length > 0) {
    const rows = Object.entries(lighthouse).map(([url, scores]) => ({ url, ...(asObject(scores) ?? {}) }));
    lines.push("lighthouse", table(rows, ["url", "performance", "seo", "accessibility", "best_practices"]));
  }
  return lines.join("\n");
}
