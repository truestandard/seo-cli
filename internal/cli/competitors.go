package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func competitorsCmd() *cobra.Command {
	c := &cobra.Command{Use: "competitors", Short: "watch competitors: what they publish and which searches they gain"}
	c.AddCommand(competitorsWatchCmd())
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
