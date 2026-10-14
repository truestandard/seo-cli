package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func planCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "plan",
		Short: "plan this cycle's fixes, refreshes, new pages and free tools, and what needs a person (paid, cents)",
		Args:  maxArgs(0, "seo plan [--articles N] [--latest]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			var data any
			if latest, _ := cmd.Flags().GetBool("latest"); latest {
				data, err = client.Get(api.ProjectPath(slug, "/publish_plan"), nil)
			} else {
				data, err = client.Post(api.ProjectPath(slug, "/publish_plan"), body(map[string]any{"articles": optInt(cmd, "articles")}))
			}
			if err != nil {
				return err
			}
			output.Render(data, func() string { return planLines(data) })
			return nil
		},
	}
	c.Flags().Int("articles", 0, "articles this cycle (default: the project's articles_per_month spread over the month)")
	c.Flags().Bool("latest", false, "print the last saved plan and spend nothing")
	return c
}

func planLines(data any) string {
	if table.IsEstimate(data) {
		return table.EstimateLine(data)
	}
	root := table.AsObject(data)
	if root["status"] != "planned" {
		return fmt.Sprintf("publish plan: %v", root["status"])
	}
	lines := []string{}
	if focus := table.AsObject(root["focus"]); focus != nil {
		lines = append(lines, fmt.Sprintf("focus: %v on %v until %v", focus["keyword"], focus["page"], focus["until"]))
	}
	if fixes := table.AsRows(root["fixes"]); len(fixes) > 0 {
		lines = append(lines, "site fixes", table.Table(fixes, []string{"severity", "rule", "count", "new", "how_to_fix"}, nil))
	}
	if refreshes := table.AsRows(root["refreshes"]); len(refreshes) > 0 {
		lines = append(lines, "refreshes", table.Table(refreshes, []string{"source", "keyword", "page", "position", "value"}, nil))
		var links []table.Row
		for _, refresh := range refreshes {
			for _, source := range table.AsRows(refresh["link_sources"]) {
				links = append(links, table.Row{"page": refresh["page"], "link_from": source["url"]})
			}
		}
		if len(links) > 0 {
			lines = append(lines, "add internal links", table.Table(links, []string{"page", "link_from"}, nil))
		}
	}
	pages := table.AsObject(root["new_pages"])
	for _, lane := range []string{"articles", "comparisons", "free_tools"} {
		if rows := table.AsRows(pages[lane]); len(rows) > 0 {
			lines = append(lines, lane, table.Table(rows, []string{"keyword", "track", "format", "floor", "expected_clicks", "drafting"}, nil))
		}
	}
	if human := table.AsRows(root["needs_human"]); len(human) > 0 {
		lines = append(lines, "needs a person", table.Table(human, []string{"kind", "keyword", "human_reason"}, nil))
	}
	lines = append(lines, fmt.Sprintf("deferred: %d · cost %s", len(table.AsRows(root["deferred"])), table.Money(root["cost"])))
	return joinLines(lines, "\n")
}
