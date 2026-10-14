package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func competitorsCmd() *cobra.Command {
	c := &cobra.Command{Use: "competitors", Short: "watch and study competitors: what they publish, where their visits come from, which searches they gain"}
	c.AddCommand(competitorsWatchCmd())
	c.AddCommand(competitorsStudyCmd())
	return c
}

func competitorsWatchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "watch",
		Short: "read the latest competitor watch; --run takes a snapshot now (paid, about $0.08 per competitor)",
		Args:  exactArgs(0, "seo competitors watch [--run] [--dry-run]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			path := api.ProjectPath(slug, "/competitor_watch")
			var data any
			if optTrue(cmd, "run") != nil || flagDryRun {
				data, err = client.Post(path, body(map[string]any{}))
			} else {
				data, err = client.Get(path, nil)
			}
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return joinLines(competitorWatchLines(table.AsObject(data)), "\n")
			})
			return nil
		},
	}
	c.Flags().Bool("run", false, "take a snapshot now instead of reading the latest one")
	return c
}

func competitorWatchLines(root table.Row) []string {
	if root["status"] == "none" {
		return []string{"no competitor snapshot yet: seo competitors watch --run"}
	}
	lines := []string{fmt.Sprintf("competitor watch %v on %v, cost %s", root["status"], root["taken_on"], table.Money(root["cost"]))}
	var rows []table.Row
	var pages []table.Row
	for _, competitor := range table.AsRows(root["competitors"]) {
		newPages := table.AsRows(competitor["new_pages"])
		competing := 0
		for _, page := range newPages {
			if page["relevance"] == "competes_for_our_buyers" {
				competing++
				pages = append(pages, table.Row{"domain": competitor["domain"], "url": page["url"], "title": page["title"]})
			}
		}
		rows = append(rows, table.Row{
			"domain": competitor["domain"], "baseline": competitor["baseline"], "new_pages": len(newPages), "competing": competing,
			"removed": competitor["removed_pages_count"], "gained": len(table.AsRows(competitor["keywords_gained"])),
			"lost": len(table.AsRows(competitor["keywords_lost"])), "error": competitor["error"],
		})
	}
	if len(rows) > 0 {
		lines = append(lines, table.Table(rows, []string{"domain", "baseline", "new_pages", "competing", "removed", "gained", "lost", "error"}, nil))
	}
	if len(pages) > 0 {
		lines = append(lines, "new pages that compete for your buyers", table.Table(pages, []string{"domain", "url", "title"}, nil))
	}
	if opportunities := table.AsRows(root["opportunities"]); len(opportunities) > 0 {
		lines = append(lines, "opportunities", table.Table(opportunities, []string{"keyword", "volume", "kd", "competitor", "their_position", "relevance", "tracked", "our_position"}, nil))
	} else {
		lines = append(lines, "no opportunities this week")
	}
	if run := table.AsObject(root["run"]); run != nil {
		lines = append(lines, fmt.Sprintf("spent %s of a %s cap", table.Money(run["spent"]), table.Money(run["cap"])))
	}
	return lines
}

func competitorsStudyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "study <domain>",
		Short: "study a competitor's pages in every market by section (paid, about $0.13; reused free for 30 days); --section drills into one (about $0.04)",
		Args:  exactArgs(1, "seo competitors study <domain> [--section /templates/] [--force] [--latest] [--dry-run]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			section, _ := cmd.Flags().GetString("section")
			var data any
			if optTrue(cmd, "latest") != nil {
				query := api.Query{"domain": args[0]}
				if section != "" {
					query["path"] = section
				}
				data, err = client.Get(api.ProjectPath(slug, "/competitor_study"), query)
			} else if section != "" {
				data, err = client.Post(api.ProjectPath(slug, "/competitor_study/section"), body(map[string]any{"domain": args[0], "path": section, "force": optTrue(cmd, "force")}))
			} else {
				data, err = client.Post(api.ProjectPath(slug, "/competitor_study"), body(map[string]any{"domain": args[0], "force": optTrue(cmd, "force")}))
			}
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return joinLines(competitorStudyLines(table.AsObject(data)), "\n")
			})
			return nil
		},
	}
	c.Flags().String("section", "", "drill into one section from the study, such as /templates/ or es.example.com")
	c.Flags().Bool("force", false, "run again even when a study under 30 days old exists")
	c.Flags().Bool("latest", false, "read the latest stored study without running one")
	return c
}

func competitorStudyLines(root table.Row) []string {
	if root["status"] == "none" {
		return []string{fmt.Sprintf("no study for %v yet: seo competitors study %v", root["domain"], root["domain"])}
	}
	spent := "cost " + table.Money(root["cost"])
	if root["cached"] == true {
		spent = fmt.Sprintf("reused from %v, nothing spent", root["studied_at"])
	}
	if keywords := table.AsRows(root["keywords"]); root["keywords"] != nil {
		buyers := 0
		for _, row := range keywords {
			if row["keep"] == true {
				buyers++
			}
		}
		return []string{
			fmt.Sprintf("%v %v (%v, %v markets): %d of %v searches, %d for your buyers, %s", root["domain"], root["path"], root["pattern"], root["markets"], len(keywords), root["total_count"], buyers, spent),
			table.Table(keywords, []string{"keyword", "position", "volume", "etv", "kd", "relevance", "keep", "url"}, map[string]string{"etv": "visits", "kd": "difficulty", "keep": "buyers"}),
		}
	}
	sections := table.AsRows(root["sections"])
	for _, row := range sections {
		if share, ok := row["share"].(float64); ok {
			row["share"] = fmt.Sprintf("%.1f%%", share*100)
		}
	}
	lines := []string{
		fmt.Sprintf("%v: %v pages in %d sections, about %v visits a month in every market, %s", root["domain"], root["pages_read"], len(sections), root["etv"], spent),
		table.Table(sections, []string{"path", "pattern", "locale", "pages", "etv", "share", "new_ranking_pages"}, map[string]string{"etv": "visits", "new_ranking_pages": "new"}),
	}
	if gaps := table.AsRows(root["gaps"]); len(gaps) > 0 {
		lines = append(lines, "page types they rank with and you lack", table.Table(gaps, []string{"pattern", "their_pages", "their_etv", "their_share", "sections"}, map[string]string{"their_etv": "their_visits"}))
	} else {
		lines = append(lines, "no gaps: you have each page type that brings them visits")
	}
	if pages := table.AsRows(root["new_pages"]); len(pages) > 0 {
		if len(pages) > 15 {
			pages = pages[:15]
		}
		lines = append(lines, "pages that started to rank", table.Table(pages, []string{"url", "etv", "keywords", "section", "first_seen"}, map[string]string{"etv": "visits"}))
	}
	return lines
}
