package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func auditCmd() *cobra.Command {
	c := &cobra.Command{Use: "audit", Short: "site audits: own crawler over the sitemap, key pages and retired paths"}
	run := auditRunCmd()
	c.RunE = run.RunE
	c.Args = exactArgs(0, "seo audit [run | show <id> | list]")
	c.Flags().AddFlagSet(run.Flags())
	c.AddCommand(run, auditShowCmd(), auditListCmd())
	return c
}

func auditRunCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "run",
		Short: "enqueue a site audit (the crawl is free; --lighthouse is paid)",
		Args:  exactArgs(0, "seo audit run [--lighthouse] [--pages <n>] [--max-pages <n>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/site_audits"), body(map[string]any{
				"lighthouse": optTrue(cmd, "lighthouse"), "pages": optInt(cmd, "pages"), "max_pages": optInt(cmd, "max-pages"),
			}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				root := table.AsObject(data)
				if table.IsEstimate(data) && root["status"] == nil {
					return table.EstimateLine(data)
				}
				id := table.Cell(root["run_id"])
				return "site audit run " + id + " enqueued: seo audit show " + id
			})
			return nil
		},
	}
	c.Flags().Bool("lighthouse", false, "add Lighthouse mobile scores for the first pages")
	c.Flags().Int("pages", 0, "pages to run Lighthouse on (default 20)")
	c.Flags().Int("max-pages", 0, "sitemap URLs to crawl (default 500)")
	return c
}

func auditShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "show a site audit run: summary, issues, pages",
		Args:  exactArgs(1, "seo audit show <id>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/site_audits/"+args[0]), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderAudit(data) })
			return nil
		},
	}
}

func auditListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list site audit runs, newest first",
		Args:  exactArgs(0, "seo audit list [--limit <n>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			query := api.Query{}
			if limit := optInt(cmd, "limit"); limit != nil {
				query["limit"] = table.Cell(limit)
			}
			data, err := client.Get(api.ProjectPath(slug, "/site_audits"), query)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				var rows []table.Row
				for _, run := range table.AsRows(data, "site_audit_runs") {
					counts := table.AsObject(run["issue_counts"])
					rows = append(rows, table.Row{
						"id": run["id"], "status": run["status"], "pages": run["pages_count"],
						"critical": counts["critical"], "warning": counts["warning"], "info": counts["info"],
						"cost": run["cost"], "created_at": run["created_at"],
					})
				}
				if len(rows) == 0 {
					return "no site audit runs"
				}
				return table.Table(rows, []string{"id", "status", "pages", "critical", "warning", "info", "cost", "created_at"}, nil)
			})
			return nil
		},
	}
	c.Flags().Int("limit", 0, "runs to list (default 20)")
	return c
}

func renderAudit(data any) string {
	run := table.First(table.AsObject(data), "site_audit_run")
	summary := table.AsObject(run["summary"])
	lines := []string{table.KeyValueBlock(fieldsOf(run, "id", "status", "pages_count", "issue_counts", "cost", "started_at", "finished_at", "error"))}
	if keyPages := table.AsObject(summary["key_pages"]); keyPages != nil {
		lines = append(lines, "key pages "+table.SummaryOf(keyPages))
	}
	if retired := table.AsObject(summary["retired"]); len(retired) > 0 {
		lines = append(lines, "retired "+table.SummaryOf(retired))
	}
	if issues := table.AsRows(run["issues"]); len(issues) > 0 {
		lines = append(lines, "issues", table.Table(issues, []string{"severity", "rule", "url", "detail"}, nil))
	}
	if lighthouse := table.AsObject(summary["lighthouse"]); len(lighthouse) > 0 {
		var rows []table.Row
		for _, pair := range pairsOf(lighthouse) {
			row := table.Row{"url": pair.Key}
			for key, value := range table.AsObject(pair.Value) {
				row[key] = value
			}
			rows = append(rows, row)
		}
		lines = append(lines, "lighthouse", table.Table(rows, []string{"url", "performance", "seo", "accessibility", "best_practices"}, nil))
	}
	return joinLines(lines, "\n")
}

func gscCmd() *cobra.Command {
	c := &cobra.Command{Use: "gsc", Short: "Search Console: import an export or read the queries close to page one"}
	c.AddCommand(gscImportCmd(), gscStrikingCmd())
	return c
}

func gscImportCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "import <csv> --dimension query|page --start YYYY-MM-DD --end YYYY-MM-DD",
		Short: "import a Search Console export (Top queries or Top pages)",
		Args:  exactArgs(1, "seo gsc import <csv> --dimension query|page --start <date> --end <date>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			dimension := flagString(cmd, "dimension")
			if dimension != "query" && dimension != "page" {
				return usageError("--dimension must be query or page")
			}
			start, end := flagString(cmd, "start"), flagString(cmd, "end")
			if start == "" || end == "" {
				return usageError("--start and --end are required")
			}
			csv, err := os.ReadFile(args[0])
			if err != nil {
				return usageError(err.Error())
			}
			data, err := client.Post(api.ProjectPath(slug, "/gsc_import"), map[string]any{
				"dimension": dimension, "range_start": start, "range_end": end, "csv": string(csv),
			})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				return table.SummaryLine([]table.Pair{{Key: "imported", Value: table.AsObject(data)["imported"]}, {Key: "dimension", Value: dimension}})
			})
			return nil
		},
	}
	c.Flags().String("dimension", "", "query or page")
	c.Flags().String("start", "", "range start YYYY-MM-DD")
	c.Flags().String("end", "", "range end YYYY-MM-DD")
	return c
}

func gscStrikingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "striking",
		Short: "queries at positions 8 to 20 by impressions",
		Args:  exactArgs(0, "seo gsc striking"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/gsc/striking_distance"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "queries", "striking_distance")
				if len(rows) == 0 {
					return "no queries close to page one"
				}
				return table.Table(rows, []string{"query", "position", "impressions", "clicks", "ctr"}, nil)
			})
			return nil
		},
	}
}

func shipCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ship <url>",
		Short: "record a shipped page and verify it answers 200 with an h1",
		Args:  exactArgs(1, "seo ship <url> [--keyword] [--track] [--note]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/ship"), compact(map[string]any{
				"url": args[0], "keyword": optString(cmd, "keyword"), "track": optString(cmd, "track"), "note": optString(cmd, "note"),
			}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				event := table.First(table.AsObject(data), "ship_event", "ship")
				url := event["url"]
				if url == nil {
					url = args[0]
				}
				keyword := event["keyword"]
				if keyword == nil {
					keyword = optString(cmd, "keyword")
				}
				return table.SummaryLine([]table.Pair{
					{Key: "url", Value: url}, {Key: "verified", Value: event["verified"]}, {Key: "http_status", Value: event["http_status"]},
					{Key: "h1", Value: event["h1"]}, {Key: "keyword", Value: keyword},
				})
			})
			return nil
		},
	}
	c.Flags().String("keyword", "", "keyword the page targets")
	c.Flags().String("track", "", "track label")
	c.Flags().String("note", "", "free-text note")
	return c
}

var experimentColumns = []string{"id", "shipped_on", "page", "change", "hypothesis", "outcome", "outcome_on"}

func experimentsCmd() *cobra.Command {
	c := &cobra.Command{Use: "experiments", Short: "on-page experiments and their outcomes"}
	list := experimentsListCmd()
	c.RunE = list.RunE
	c.Args = exactArgs(0, "seo experiments [list | add | outcome]")
	c.AddCommand(list, experimentsAddCmd(), experimentsOutcomeCmd())
	return c
}

func experimentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list experiments",
		Args:  exactArgs(0, "seo experiments list"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/experiments"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "experiments")
				if len(rows) == 0 {
					return "no experiments"
				}
				return table.Table(rows, experimentColumns, nil)
			})
			return nil
		},
	}
}

func experimentsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <change...>",
		Short: "record an experiment (what changed)",
		Args:  minArgs(1, "seo experiments add <change...> [--page --hypothesis --keyword --shipped-on]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/experiments"), compact(map[string]any{
				"change": joinArgs(args), "page": optString(cmd, "page"), "hypothesis": optString(cmd, "hypothesis"),
				"keyword": optString(cmd, "keyword"), "shipped_on": optString(cmd, "shipped-on"),
			}))
			if err != nil {
				return err
			}
			output.Render(data, func() string { return table.KeyValueBlock(experimentFields(data)) })
			return nil
		},
	}
	c.Flags().String("page", "", "page path the change shipped on")
	c.Flags().String("hypothesis", "", "why it should move the needle")
	c.Flags().String("keyword", "", "keyword it targets")
	c.Flags().String("shipped-on", "", "YYYY-MM-DD (default today)")
	return c
}

func experimentsOutcomeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "outcome <id> <outcome...>",
		Short: "record the outcome of an experiment",
		Args:  minArgs(2, "seo experiments outcome <id> <outcome...>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Patch(api.ProjectPath(slug, "/experiments/"+args[0]), map[string]any{"outcome": joinArgs(args[1:])})
			if err != nil {
				return err
			}
			output.Render(data, func() string { return table.KeyValueBlock(experimentFields(data)) })
			return nil
		},
	}
}

func experimentFields(data any) []table.Pair {
	return fieldsOf(table.First(table.AsObject(data), "experiment"), experimentColumns...)
}

func logCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "log",
		Short: "research log entries",
		Args:  exactArgs(0, "seo log [--kind <kind>] [--days <n>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			query := api.Query{"kind": flagString(cmd, "kind")}
			if days := optInt(cmd, "days"); days != nil {
				query["days"] = table.Cell(days)
			}
			data, err := client.Get(api.ProjectPath(slug, "/research_log"), query)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "entries", "research_log")
				if len(rows) == 0 {
					return "no log entries"
				}
				return table.Table(rows, []string{"created_at", "kind", "summary", "cost", "actor"}, nil)
			})
			return nil
		},
	}
	c.Flags().String("kind", "", "filter by kind")
	c.Flags().Int("days", 0, "look back this many days (default 30)")
	return c
}

func scoreboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scoreboard",
		Short: "targets, ranks, AI answers, shipped pages and 30-day spend in one read",
		Args:  exactArgs(0, "seo scoreboard"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/scoreboard"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				board := table.First(table.AsObject(data), "scoreboard")
				var lines []string
				if floors := table.AsObject(board["floors"]); len(floors) > 0 {
					lines = append(lines, "targets "+table.SummaryOf(floors))
				}
				if ranks := table.AsObject(board["ranks"]); ranks != nil {
					lines = append(lines, "ranks   "+table.SummaryOf(ranks))
				}
				if ai := table.AsObject(board["ai"]); ai != nil {
					lines = append(lines, "ai      "+table.SummaryOf(ai))
				}
				var spend any
				if board["spend_30d"] != nil {
					spend = table.Money(board["spend_30d"])
				}
				lines = append(lines, table.KeyValueBlock([]table.Pair{
					{Key: "ships", Value: board["ships"]}, {Key: "spend_30d", Value: spend},
					{Key: "last_rank_run_on", Value: board["last_rank_run_on"]}, {Key: "last_ai_scan_on", Value: board["last_ai_scan_on"]},
				}))
				return joinLines(lines, "\n")
			})
			return nil
		},
	}
}

func spendCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "spend",
		Short: "paid calls across providers since a date",
		Args:  exactArgs(0, "seo spend [--since 30d]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			window, err := since(flagString(cmd, "since"))
			if err != nil {
				return err
			}
			data, err := client.Get("/api/v1/spend", api.Query{"since": window})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				root := table.AsObject(data)
				var lines []string
				byProvider := table.AsRows(root["by_provider"])
				if byProvider == nil {
					byProvider = table.AsRows(root["providers"])
				}
				if len(byProvider) > 0 {
					lines = append(lines, table.Table(byProvider, nil, nil))
				}
				entries := table.AsRows(root["entries"])
				if entries == nil {
					entries = table.AsRows(root["spend_entries"])
				}
				if entries == nil {
					entries = table.AsRows(root["rows"])
				}
				if len(entries) > 0 {
					lines = append(lines, table.Table(entries, []string{"created_at", "project", "provider", "endpoint", "units", "cost"}, nil))
				}
				if root["total"] != nil {
					lines = append(lines, "total "+table.Money(root["total"])+" since "+window)
				}
				if len(lines) == 0 {
					return "no spend since " + window
				}
				return joinLines(lines, "\n")
			})
			return nil
		},
	}
	c.Flags().String("since", "30d", "7d, 30d, or YYYY-MM-DD")
	return c
}
