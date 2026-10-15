package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func trackBingCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "bing",
		Short: "Bing positions next to Google; --run checks Bing now (paid, about $0.0035 per keyword)",
		Args:  exactArgs(0, "seo track bing [--run [--wait]] [--set <name>] [--dry-run]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			path := api.ProjectPath(slug, "/bing_ranks")
			var data any
			if optTrue(cmd, "run") != nil || flagDryRun {
				data, err = client.Post(path, body(map[string]any{"set_name": optString(cmd, "set"), "wait": optTrue(cmd, "wait")}))
			} else {
				data, err = client.Get(path, api.Query{"set_name": flagString(cmd, "set")})
			}
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderBing(data) })
			return nil
		},
	}
	c.Flags().Bool("run", false, "check Bing now instead of reading the latest check")
	c.Flags().Bool("wait", false, "with --run, wait for the check instead of queueing it")
	c.Flags().String("set", "", "keyword set (default: every tracked keyword)")
	return c
}

func renderBing(data any) string {
	if table.IsEstimate(data) {
		return table.EstimateLine(data)
	}
	root := table.AsObject(data)
	if root["status"] == "enqueued" {
		return fmt.Sprintf("Bing check queued (job %v); read it with seo track bing", root["job_id"])
	}
	if root["status"] == "none" {
		return "no Bing check yet: seo track bing --run"
	}
	lines := []string{fmt.Sprintf("Bing checked %v", root["checked_on"])}
	if summary := table.AsObject(root["summary"]); summary != nil {
		lines = append(lines, table.SummaryLine(fieldsOf(summary, "checked", "bing_top10", "google_top10", "bing_ranked", "google_ranked")))
	}
	if rows := table.AsRows(root["disagreements"]); len(rows) > 0 {
		lines = append(lines, "Bing and Google disagree by 10+ positions", table.Table(rows, []string{"keyword", "google", "bing", "gap"}, nil))
	}
	if rows := table.AsRows(root["google_only"]); len(rows) > 0 {
		lines = append(lines, "in Google's top 20, not ranked on Bing (ChatGPT search reads Bing)", table.Table(rows, []string{"keyword", "google"}, nil))
	}
	if run := table.AsObject(root["run"]); run != nil {
		lines = append(lines, fmt.Sprintf("checked %v, cost %s", run["checked"], table.Money(root["cost"])))
	}
	return joinLines(lines, "\n")
}

func aiDemandCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "demand",
		Short: "AI search demand for the panel prompts; --run fetches it (paid, about $0.01 a call)",
		Args:  exactArgs(0, "seo ai demand [--run [--force]] [--set <name>] [--dry-run]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			path := api.ProjectPath(slug, "/ai_demand")
			var data any
			if optTrue(cmd, "run") != nil || flagDryRun {
				data, err = client.Post(path, body(map[string]any{"set_name": optString(cmd, "set"), "force": optTrue(cmd, "force")}))
			} else {
				data, err = client.Get(path, api.Query{"set_name": flagString(cmd, "set")})
			}
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderDemand(data) })
			return nil
		},
	}
	c.Flags().Bool("run", false, "fetch AI search volume now instead of reading the latest")
	c.Flags().Bool("force", false, "with --run, skip the 30-day reuse")
	c.Flags().String("set", "", "prompt set (default: every set)")
	return c
}

func renderDemand(data any) string {
	if table.IsEstimate(data) {
		return table.EstimateLine(data)
	}
	root := table.AsObject(data)
	rows := table.AsRows(root["prompts"])
	if len(rows) == 0 {
		return "no prompt demand yet: seo ai demand --run"
	}
	lines := []string{table.Table(rows, []string{"demand", "text", "matched", "locked"}, nil)}
	if suggestion, ok := root["suggestion"].(string); ok && suggestion != "" {
		lines = append(lines, suggestion)
	}
	return joinLines(lines, "\n")
}
