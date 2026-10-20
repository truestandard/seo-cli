package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func linksQueueCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "queue",
		Short: "link targets for every watched page and whether new pages wait for more linking sites (build is paid, cents)",
		Args:  exactArgs(0, "seo links queue [--url U] [--run] [--force] [--dry-run]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			path := api.ProjectPath(slug, "/link_queue")
			url := optString(cmd, "url")
			var data any
			if url != nil || optTrue(cmd, "run") != nil || flagDryRun {
				data, err = client.Post(path, body(map[string]any{"url": url, "force": optTrue(cmd, "force")}))
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
				return joinLines(linkQueueLines(table.AsObject(data)), "\n")
			})
			return nil
		},
	}
	c.Flags().String("url", "", "build the queue for one watched page")
	c.Flags().Bool("run", false, "build every page with no queue or one older than 28 days, inside the plan spend cap")
	c.Flags().Bool("force", false, "build again even when the queue is under 28 days old")
	return c
}

func linkQueueLines(root table.Row) []string {
	lines := []string{}
	if pause := table.AsObject(root["pause"]); pause != nil {
		lines = append(lines, fmt.Sprint(pause["reason"]))
	}
	if queue := table.AsObject(root["link_queue"]); queue != nil {
		spent := "cost " + table.Money(root["cost"])
		if queue["cached"] == true {
			spent = fmt.Sprintf("reused %v, nothing spent", queue["built_on"])
		}
		lines = append(lines, fmt.Sprintf("%v (%s, %v link items opened)", root["url"], spent, orZero(queue["opened_items"])))
		if internal := table.AsRows(queue["internal"]); len(internal) > 0 {
			lines = append(lines, "link from your pages", table.Table(internal, []string{"url", "title", "links_now"}, nil))
		}
		if external := table.AsRows(queue["external"]); len(external) > 0 {
			lines = append(lines, "outside sites to ask (a person sends any pitch)", table.Table(external, []string{"domain", "reason"}, nil))
		}
		return lines
	}
	if built, ok := root["built"].([]any); ok {
		lines = append(lines, fmt.Sprintf("built %d, skipped at the cap %d, failed %d (cost %s)", len(built), listLen(root["skipped_for_budget"]), listLen(root["failed"]), table.Money(root["cost"])))
		if rows := table.AsRows(built); len(rows) > 0 {
			lines = append(lines, table.Table(rows, []string{"url", "internal", "external", "opened_items"}, nil))
		}
		return lines
	}
	watches := table.AsRows(root["watches"])
	lines = append(lines, fmt.Sprintf("%v pages have link plans, %v due; open link items: %v", orZero(root["built"]), orZero(root["due"]), orZero(root["open_link_items"])))
	rows := make([]table.Row, 0, len(watches))
	for _, watch := range watches {
		queue := table.AsObject(watch["link_queue"])
		sources := 0
		for _, source := range table.AsRows(queue["internal"]) {
			if source["links_now"] != true {
				sources++
			}
		}
		rows = append(rows, table.Row{"url": watch["url"], "state": watch["state"], "built_on": queue["built_on"], "sources": sources, "external": len(table.AsRows(queue["external"]))})
	}
	if len(rows) > 0 {
		lines = append(lines, table.Table(rows, []string{"url", "state", "built_on", "sources", "external"}, nil))
	}
	return lines
}

func orZero(value any) any {
	if value == nil {
		return 0
	}
	return value
}

func listLen(value any) int {
	list, _ := value.([]any)
	return len(list)
}
