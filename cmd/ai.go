package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func promptsCmd() *cobra.Command {
	c := &cobra.Command{Use: "prompts", Short: "AI answer prompt sets"}
	list := promptsListCmd()
	c.RunE = list.RunE
	c.Args = exactArgs(0, "seo prompts [list | add]")
	c.Flags().AddFlagSet(list.Flags())
	c.AddCommand(list, promptsAddCmd())
	return c
}

func promptsListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list prompts",
		Args:  exactArgs(0, "seo prompts list [--set <name>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/ai_prompts"), api.Query{"set": flagString(cmd, "set")})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "prompts", "ai_prompts")
				if len(rows) == 0 {
					return "no prompts"
				}
				return table.Table(rows, []string{"id", "set_name", "locked", "text"}, map[string]string{"set_name": "set"})
			})
			return nil
		},
	}
	c.Flags().String("set", "", "filter by set name")
	return c
}

func promptsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [texts...]",
		Short: "add prompts (one per argument, or one per line with --file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			texts := append([]string{}, args...)
			if file := flagString(cmd, "file"); file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return usageError(err.Error())
				}
				for _, line := range strings.Split(string(data), "\n") {
					if trimmed := strings.TrimSpace(line); trimmed != "" {
						texts = append(texts, trimmed)
					}
				}
			}
			if len(texts) == 0 {
				return usageError("no prompts given: pass them as arguments or with --file")
			}
			var prompts []any
			for _, text := range texts {
				prompts = append(prompts, map[string]any{"text": text, "set_name": optString(cmd, "set"), "locked": optBool(cmd, "locked")})
			}
			data, err := client.Post(api.ProjectPath(slug, "/ai_prompts"), map[string]any{"prompts": compactAll(prompts)})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "prompts", "ai_prompts")
				if len(rows) == 0 {
					return "added " + pluralize(len(texts), "prompt")
				}
				return table.Table(rows, []string{"id", "set_name", "locked", "text"}, map[string]string{"set_name": "set"})
			})
			return nil
		},
	}
	c.Flags().String("set", "", "set name (default guarantee)")
	c.Flags().Bool("locked", false, "lock the prompts as a measurement contract")
	c.Flags().String("file", "", "read prompts from a file, one per line")
	return c
}

func aiCmd() *cobra.Command {
	c := &cobra.Command{Use: "ai", Short: "AI answer scans: run one, read the results, or check a scan"}
	run := aiRunCmd()
	c.RunE = run.RunE
	c.Args = exactArgs(0, "seo ai [run | results | status <id>]")
	c.Flags().AddFlagSet(run.Flags())
	c.AddCommand(run, aiResultsCmd(), aiStatusCmd())
	return c
}

func aiRunCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "run",
		Short: "run the AI prompts across the pinned models (paid)",
		Args:  exactArgs(0, "seo ai run [--set <name>] [--runs <n>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/ai_scans"), body(map[string]any{"set_name": optString(cmd, "set"), "runs_per_cell": optInt(cmd, "runs")}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return table.KeyValueBlock(scanFields(data))
			})
			return nil
		},
	}
	c.Flags().String("set", "", "prompt set (default guarantee)")
	c.Flags().Int("runs", 0, "runs per prompt and model pair (default 1)")
	return c
}

func aiResultsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "results",
		Short: "AI answer report and changes since a date",
		Args:  exactArgs(0, "seo ai results [--since 30d] [--set guarantee]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			window, err := since(flagString(cmd, "since"))
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/ai_visibility"), api.Query{"since": window, "set": flagString(cmd, "set")})
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderVisibility(data) })
			return nil
		},
	}
	c.Flags().String("since", "30d", "7d, 30d, or YYYY-MM-DD")
	c.Flags().String("set", "guarantee", "prompt set")
	return c
}

func aiStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "show an AI scan",
		Args:  exactArgs(1, "seo ai status <id>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/ai_scans/"+args[0]), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return table.KeyValueBlock(scanFields(data)) })
			return nil
		},
	}
}

func scanFields(data any) []table.Pair {
	scan := table.First(table.AsObject(data), "ai_scan", "scan")
	return fieldsOf(scan, "id", "status", "run_on", "set_name", "models", "runs_per_cell", "cost", "summary")
}

func renderVisibility(data any) string {
	root := table.AsObject(data)
	var lines []string
	if report := table.AsObject(root["report"]); report != nil {
		lines = append(lines, table.SummaryLine(fieldsOf(report, "cells", "ok", "errors", "named", "cited", "queries_hit", "queries_always")))
		if perEngine := table.AsObject(report["per_engine"]); perEngine != nil {
			var rows []table.Row
			for _, pair := range pairsOf(perEngine) {
				row := table.Row{"engine": pair.Key}
				for key, value := range table.AsObject(pair.Value) {
					row[key] = value
				}
				rows = append(rows, row)
			}
			lines = append(lines, table.Table(rows, []string{"engine", "cited", "with_urls", "blank"}, nil))
		}
		if rows := table.AsRows(report["domains_top"]); len(rows) > 0 {
			lines = append(lines, table.Table(rows, nil, nil))
		}
		if rivals := table.AsObject(report["rival_share"]); rivals != nil {
			lines = append(lines, "rivals "+table.SummaryOf(rivals))
		}
	}
	if deltas := table.AsObject(root["deltas"]); deltas != nil {
		if totals := table.AsObject(deltas["totals"]); totals != nil {
			lines = append(lines, "totals "+table.SummaryOf(totals))
		}
		if rows := table.AsRows(deltas["rows"]); len(rows) > 0 {
			lines = append(lines, table.Table(rows, nil, nil))
		}
	} else if rows := table.AsRows(root["deltas"]); len(rows) > 0 {
		lines = append(lines, table.Table(rows, nil, nil))
	}
	if len(lines) == 0 {
		return "no AI scan in this window"
	}
	return joinLines(lines, "\n")
}

func joinLines(lines []string, separator string) string {
	var kept []string
	for _, line := range lines {
		if line != "" {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, separator)
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
