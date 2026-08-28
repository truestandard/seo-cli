package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func researchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "research <seeds...>",
		Short: "keyword research from seed terms (paid)",
		Args:  minArgs(1, "seo research <seeds...> [--limit --max-kd --keywords]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			fields := map[string]any{"limit": optInt(cmd, "limit"), "max_kd": optInt(cmd, "max-kd")}
			if exact, _ := cmd.Flags().GetBool("keywords"); exact {
				fields["keywords"] = args
			} else {
				fields["seeds"] = args
			}
			data, err := client.Post(api.ProjectPath(slug, "/research"), body(fields))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				rows := table.AsRows(data, "keywords", "suggestions")
				if len(rows) == 0 {
					return "no keywords returned"
				}
				lines := []string{table.Table(rows, []string{"keyword", "volume", "kd", "cpc", "intent", "yoy"}, nil)}
				cost := table.Pick(data, "spend")["cost"]
				if cost == nil {
					cost = table.AsObject(data)["cost"]
				}
				if cost != nil {
					lines = append(lines, "cost "+table.Money(cost))
				}
				return joinLines(lines, "\n")
			})
			return nil
		},
	}
	c.Flags().Int("limit", 0, "max suggestions per seed (default 40)")
	c.Flags().Int("max-kd", 0, "max keyword difficulty (default 30)")
	c.Flags().Bool("keywords", false, "treat the arguments as exact keywords for an overview instead of seeds")
	return c
}

func serpCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "serp <keyword>",
		Short: "one live Google results page for a keyword (paid)",
		Args:  exactArgs(1, "seo serp <keyword> [--depth <n>]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/serp"), body(map[string]any{"keyword": args[0], "depth": optInt(cmd, "depth")}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				result := table.First(table.AsObject(data), "result")
				lines := []string{table.SummaryLine([]table.Pair{
					{Key: "keyword", Value: args[0]}, {Key: "position", Value: result["position"]},
					{Key: "rank_absolute", Value: result["rank_absolute"]}, {Key: "url", Value: result["url"]},
					{Key: "path_match", Value: result["path_match"]}, {Key: "features", Value: result["features"]},
				})}
				if rows := table.AsRows(result["top_domains"]); len(rows) > 0 {
					lines = append(lines, table.Table(rows, []string{"rank", "domain", "url", "title"}, nil))
				}
				return joinLines(lines, "\n")
			})
			return nil
		},
	}
	c.Flags().Int("depth", 0, "results depth (default 100)")
	return c
}

func trackCmd() *cobra.Command {
	c := &cobra.Command{Use: "track", Short: "rank checks: start one or read its status"}
	run := trackRunCmd()
	c.RunE = run.RunE
	c.Args = exactArgs(0, "seo track [run | status <id>]")
	c.Flags().AddFlagSet(run.Flags())
	c.AddCommand(run, trackStatusCmd())
	return c
}

func trackRunCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "run",
		Short: "start a rank check (paid)",
		Args:  exactArgs(0, "seo track run [--live | --scheduled] [--set <name>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			live, _ := cmd.Flags().GetBool("live")
			scheduled, _ := cmd.Flags().GetBool("scheduled")
			if live && scheduled {
				return usageError("pass either --live or --scheduled, not both")
			}
			mode := "scheduled"
			if live {
				mode = "live"
			}
			data, err := client.Post(api.ProjectPath(slug, "/rank_runs"), body(map[string]any{"mode": mode, "set_name": optString(cmd, "set")}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return table.KeyValueBlock(rankRunFields(data))
			})
			return nil
		},
	}
	c.Flags().Bool("live", false, "live search calls, completes now")
	c.Flags().Bool("scheduled", false, "standard queue, polled later (default)")
	c.Flags().String("set", "", "keyword set (default guarantee)")
	return c
}

func trackStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "show a rank check",
		Args:  exactArgs(1, "seo track status <id>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/rank_runs/"+args[0]), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return table.KeyValueBlock(rankRunFields(data)) })
			return nil
		},
	}
}

func rankRunFields(data any) []table.Pair {
	run := table.First(table.AsObject(data), "rank_run")
	return fieldsOf(run, "id", "mode", "status", "checked_on", "keyword_count", "completed_count", "cost", "summary")
}

func fieldsOf(object table.Row, keys ...string) []table.Pair {
	pairs := make([]table.Pair, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, table.Pair{Key: key, Value: object[key]})
	}
	return pairs
}

func ranksCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ranks",
		Short: "rank changes for a set since a date",
		Args:  exactArgs(0, "seo ranks [--since 7d] [--set guarantee]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			window, err := since(flagString(cmd, "since"))
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/ranks"), api.Query{"since": window, "set": flagString(cmd, "set")})
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderRanks(data) })
			return nil
		},
	}
	c.Flags().String("since", "7d", "7d, 30d, or YYYY-MM-DD")
	c.Flags().String("set", "guarantee", "keyword set")
	return c
}

func renderRanks(data any) string {
	var rows []table.Row
	for _, row := range table.AsRows(data, "rows") {
		rows = append(rows, table.Row{
			"keyword": row["keyword"], "pos": row["position"], "prev": row["previous"],
			"delta": signed(row["delta"]), "band": row["band_change"], "url": row["url"],
		})
	}
	lines := []string{"no ranked keywords in this window"}
	if len(rows) > 0 {
		lines[0] = table.Table(rows, []string{"keyword", "pos", "prev", "delta", "band", "url"}, nil)
	}
	if summary := table.Pick(data, "summary"); summary != nil {
		lines = append(lines, table.SummaryLine(fieldsOf(summary, "top10", "top20", "top100", "unranked", "avg_position", "floor_target", "floor_met")))
	}
	return joinLines(lines, "\n")
}

func signed(value any) any {
	number, ok := value.(float64)
	if !ok {
		return value
	}
	if number > 0 {
		return "+" + strconv.FormatFloat(number, 'f', -1, 64)
	}
	return strconv.FormatFloat(number, 'f', -1, 64)
}

func floorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "floor <keywords...>",
		Short: "win check: the weakest page-one site's linking sites against your own (paid)",
		Args:  minArgs(1, "seo floor <keywords...>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/floor"), body(map[string]any{"keywords": args}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				rows := table.AsRows(data, "probes", "floor_probes")
				if len(rows) == 0 {
					return "no probes returned"
				}
				return table.Table(rows,
					[]string{"keyword_text", "verdict", "own_referring_domains", "floor_referring_domains", "weakest_domain", "ratio"},
					map[string]string{"keyword_text": "keyword", "own_referring_domains": "own_rds", "floor_referring_domains": "floor_rds"})
			})
			return nil
		},
	}
}

func backlinksCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "backlinks [domain]",
		Short: "backlink snapshot for a domain, with monthly history and top linking sites on request (paid)",
		Args:  maxArgs(1, "seo backlinks [domain] [--history --months --rows]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			fields := map[string]any{"history": optTrue(cmd, "history"), "months": optInt(cmd, "months"), "rows": optInt(cmd, "rows")}
			if len(args) == 1 {
				fields["domain"] = args[0]
			}
			data, err := client.Post(api.ProjectPath(slug, "/backlinks"), body(fields))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return renderBacklinks(data)
			})
			return nil
		},
	}
	c.Flags().Bool("history", false, "add 12 months of backlinks, linking sites, new and lost")
	c.Flags().Int("months", 0, "months of history (default 12)")
	c.Flags().Int("rows", 0, "list the top N linking sites by backlinks")
	return c
}

func renderBacklinks(data any) string {
	root := table.AsObject(data)
	snapshot := table.First(root, "snapshot", "backlink_snapshot")
	lines := []string{table.KeyValueBlock(fieldsOf(snapshot, "domain", "measured_on", "referring_domains", "backlinks", "rank", "spam_score"))}
	if rows := table.AsRows(root["history"]); len(rows) > 0 {
		lines = append(lines, "history", table.Table(rows, []string{"month", "backlinks", "referring_domains", "new", "lost"}, nil))
	}
	if rows := table.AsRows(root["referring_domains"]); len(rows) > 0 {
		lines = append(lines, "linking sites", table.Table(rows, []string{"domain", "rank", "backlinks", "dofollow", "first_seen"}, nil))
	}
	if root["cost"] != nil {
		lines = append(lines, "cost "+table.Money(root["cost"]))
	}
	return joinLines(lines, "\n")
}

func domainCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "domain [domain]",
		Short: "domain overview: organic traffic, keyword count, top keywords and pages (paid, cached 12h)",
		Args:  maxArgs(1, "seo domain [domain] [--limit --force]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			fields := map[string]any{"limit": optInt(cmd, "limit"), "force": optTrue(cmd, "force")}
			if len(args) == 1 {
				fields["domain"] = args[0]
			}
			data, err := client.Post(api.ProjectPath(slug, "/domain_overview"), body(fields))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return renderOverview(data)
			})
			return nil
		},
	}
	c.Flags().Int("limit", 0, "top keywords to fetch (default 100)")
	c.Flags().Bool("force", false, "bypass the 12h cache")
	return c
}

func renderOverview(data any) string {
	root := table.AsObject(data)
	lines := []string{table.SummaryLine(fieldsOf(root, "domain", "organic_traffic", "organic_keywords", "fetched_at"))}
	if rows := table.AsRows(root["top_keywords"]); len(rows) > 0 {
		lines = append(lines, "top keywords", table.Table(rows, []string{"keyword", "position", "volume", "traffic", "cpc", "url"}, nil))
	}
	if rows := table.AsRows(root["top_pages"]); len(rows) > 0 {
		lines = append(lines, "top pages", table.Table(rows, []string{"url", "traffic", "keywords"}, nil))
	}
	lines = append(lines, costLine(root))
	return joinLines(lines, "\n")
}

func costLine(root table.Row) string {
	if cached, _ := root["cached"].(bool); cached {
		return "cached " + table.Cell(root["cached_at"]) + ", nothing spent"
	}
	return "cost " + table.Money(root["cost"])
}

func mentionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "mentions",
		Short: "brand mentions in ChatGPT and Google AI answers (paid, cached 24h)",
		Args:  exactArgs(0, "seo mentions [--brand <name>] [--competitor <name>]... [--force]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			competitors, _ := cmd.Flags().GetStringArray("competitor")
			fields := map[string]any{"brand": optString(cmd, "brand"), "force": optTrue(cmd, "force")}
			if len(competitors) > 0 {
				fields["competitors"] = competitors
			}
			data, err := client.Post(api.ProjectPath(slug, "/llm_mentions"), body(fields))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return renderMentions(data)
			})
			return nil
		},
	}
	c.Flags().String("brand", "", "brand name (default: the project name)")
	c.Flags().StringArray("competitor", nil, "competitor brand, repeatable")
	c.Flags().Bool("force", false, "bypass the 24h cache")
	return c
}

func renderMentions(data any) string {
	root := table.AsObject(data)
	platforms := table.AsObject(root["platforms"])
	lines := []string{table.SummaryLine(fieldsOf(root, "brand", "fetched_at"))}
	var rows []table.Row
	for _, pair := range pairsOf(platforms) {
		details := table.AsObject(pair.Value)
		row := table.Row{"platform": pair.Key, "mentions": details["mentions"], "ai_search_volume": details["ai_search_volume"]}
		if pages := table.AsRows(details["top_pages"]); len(pages) > 0 {
			row["top_page"] = pages[0]["url"]
		}
		rows = append(rows, row)
	}
	if len(rows) > 0 {
		lines = append(lines, table.Table(rows, []string{"platform", "mentions", "ai_search_volume", "top_page"}, nil))
	}
	if share := table.AsObject(root["share_of_voice"]); share != nil {
		lines = append(lines, "share of voice "+table.SummaryOf(share))
	}
	for _, pair := range pairsOf(platforms) {
		prompts := table.AsRows(table.AsObject(pair.Value)["sample_prompts"])
		if len(prompts) > 5 {
			prompts = prompts[:5]
		}
		if len(prompts) > 0 {
			lines = append(lines, pair.Key+" prompts", table.Table(prompts, []string{"question", "ai_search_volume", "cites_own"}, nil))
		}
	}
	lines = append(lines, costLine(root))
	return joinLines(lines, "\n")
}

func flagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}
