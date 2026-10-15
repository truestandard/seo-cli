package cli

import (
	"fmt"
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
	c.Args = exactArgs(0, "seo ai [run | results | status <id> | fixes | agents]")
	c.Flags().AddFlagSet(run.Flags())
	c.AddCommand(run, aiResultsCmd(), aiStatusCmd(), aiFixesCmd(), aiAgentsCmd())
	return c
}

const agentTaskSet = "agent_tasks"

func aiAgentsCmd() *cobra.Command {
	c := &cobra.Command{Use: "agents", Short: "agent panel: does a coding agent pick you for a task, with and without web search"}
	results := aiAgentsResultsCmd()
	c.RunE = results.RunE
	c.Args = exactArgs(0, "seo ai agents [results | add <tasks...> | suggest | run]")
	c.AddCommand(results, aiAgentsAddCmd(), aiAgentsSuggestCmd(), aiAgentsRunCmd())
	return c
}

func aiAgentsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <tasks...>",
		Short: "add agent tasks, one per argument: what a developer would ask a coding agent to do",
		Args:  minArgs(1, "seo ai agents add <tasks...> [--locked]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			var prompts []any
			for _, text := range args {
				prompts = append(prompts, map[string]any{"text": text, "set_name": agentTaskSet, "locked": optBool(cmd, "locked")})
			}
			data, err := client.Post(api.ProjectPath(slug, "/ai_prompts"), map[string]any{"prompts": compactAll(prompts)})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				return table.Table(table.AsRows(data, "prompts", "ai_prompts"), []string{"id", "locked", "text"}, nil)
			})
			return nil
		},
	}
	c.Flags().Bool("locked", false, "lock the tasks so the monthly run measures them")
	return c
}

func aiAgentsSuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: "draft candidate agent tasks from the project context (paid, about $0.02; nothing is saved)",
		Args:  exactArgs(0, "seo ai agents suggest [--dry-run]"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/agent_panel/suggest"), body(map[string]any{}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				lines := []string{"candidate agent tasks (not saved; add the real ones with seo ai agents add):"}
				tasks, _ := table.AsObject(data)["tasks"].([]any)
				for _, task := range tasks {
					lines = append(lines, fmt.Sprintf("  %v", task))
				}
				return joinLines(append(lines, "cost "+table.Money(table.AsObject(data)["cost"])), "\n")
			})
			return nil
		},
	}
}

func aiAgentsRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "run the agent panel on every agent task, with and without web search (paid, about $0.26 a task)",
		Args:  exactArgs(0, "seo ai agents run [--dry-run]"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/agent_panel"), body(map[string]any{}))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				root := table.AsObject(data)
				return fmt.Sprintf("agent panel %v (job %v), estimate %s; read it with seo ai agents results", root["status"], root["job_id"], table.Money(table.AsObject(root["estimate"])["cost"]))
			})
			return nil
		},
	}
}

func aiAgentsResultsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "results",
		Short: "the latest agent panel: picks with and without search, and the lever per task",
		Args:  exactArgs(0, "seo ai agents results"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/agent_panel"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return joinLines(agentPanelLines(table.AsObject(data)), "\n") })
			return nil
		},
	}
}

func agentPanelLines(root table.Row) []string {
	if root["status"] == "none" {
		return []string{fmt.Sprintf("no agent panel yet (%v tasks in %v): seo ai agents add \"<task>\", then seo ai agents run", root["prompts_in_set"], root["set_name"])}
	}
	ours := table.AsObject(root["ours"])
	with := table.AsObject(ours["with_search"])
	without := table.AsObject(ours["without_search"])
	lines := []string{fmt.Sprintf("agent panel %v: %v picked %v/%v with search, %v/%v without, cost %s", root["run_on"], ours["name"], with["picked"], with["cells"], without["picked"], without["cells"], table.Money(root["cost"]))}
	var rows []table.Row
	for _, prompt := range table.AsRows(root["prompts"]) {
		withSide := table.AsObject(prompt["with_search"])
		withoutSide := table.AsObject(prompt["without_search"])
		top := ""
		if picks := table.AsRows(withoutSide["picks"]); len(picks) > 0 {
			top = fmt.Sprint(picks[0]["tool"])
		}
		var installs []string
		for _, item := range table.AsRows(withoutSide["installs"]) {
			if len(installs) < 2 {
				installs = append(installs, fmt.Sprint(item["package"]))
			}
		}
		rows = append(rows, table.Row{
			"lever": prompt["lever"], "task": prompt["prompt"],
			"with_search":    fmt.Sprintf("%v/%v", withSide["ours"], withSide["cells"]),
			"without_search": fmt.Sprintf("%v/%v", withoutSide["ours"], withoutSide["cells"]),
			"top_pick":       top,
			"installs":       strings.Join(installs, ", "),
		})
	}
	if len(rows) > 0 {
		lines = append(lines, table.Table(rows, []string{"lever", "task", "with_search", "without_search", "top_pick", "installs"}, map[string]string{"top_pick": "top pick without search", "installs": "installs instead"}))
	}
	if tools := table.AsRows(root["tools"]); len(tools) > 0 {
		var shares []table.Row
		for _, tool := range tools {
			shares = append(shares, table.Row{"tool": tool["tool"], "with_search": table.AsObject(tool["with_search"])["picks"], "without_search": table.AsObject(tool["without_search"])["picks"]})
		}
		lines = append(lines, "picks by tool", table.Table(shares, []string{"tool", "with_search", "without_search"}, nil))
	}
	if installs := table.AsRows(root["installs"]); len(installs) > 0 {
		lines = append(lines, "what the agents install", table.Table(installs, []string{"package", "with_search", "without_search"}, nil))
	}
	return lines
}

func aiFixesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "fixes",
		Short: "actions from the latest AI scan, AI Overview checks and site audit",
		Args:  exactArgs(0, "seo ai fixes [--set guarantee]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/ai_fixes"), api.Query{"set": flagString(cmd, "set")})
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderFixes(data) })
			return nil
		},
	}
	c.Flags().String("set", "guarantee", "prompt set")
	return c
}

func renderFixes(data any) string {
	root := table.AsObject(data)
	rows := table.AsRows(root["actions"])
	header := fmt.Sprintf("ai fixes: %v, scan %v measured %v", root["status"], root["scan_id"], root["measured_at"])
	if len(rows) == 0 {
		return header + "\nno actions"
	}
	human := 0
	for _, row := range rows {
		if row["needs_human"] == true {
			human++
		}
	}
	header += fmt.Sprintf(", %s, %d need a person", pluralize(len(rows), "action"), human)
	return joinLines([]string{header, table.Table(rows, []string{"priority", "kind", "prompt", "keyword", "page", "engines", "needs_human"}, map[string]string{"needs_human": "person"})}, "\n")
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
		if rivals := table.AsObject(report["competitor_share_of_voice"]); len(rivals) > 0 {
			label := "rivals named"
			if report["competitor_share_of_voice_method"] == "jev_suggested" {
				label = "rivals recommended"
			}
			lines = append(lines, label+" "+table.SummaryOf(rivals))
		}
		lines = append(lines, judgmentLines(table.AsObject(report["judgment"]))...)
		lines = append(lines, accuracyLines(table.AsObject(report["accuracy"]))...)
	}
	if supplementary := table.AsObject(root["supplementary"]); supplementary != nil {
		if report := table.AsObject(supplementary["report"]); report != nil {
			lines = append(lines, "supplementary "+table.SummaryLine(fieldsOf(report, "models", "cells", "named", "cited", "queries_hit")))
		}
	}
	if overview := table.AsObject(root["ai_overview"]); overview != nil {
		if count, ok := overview["with_ai_overview"].(float64); ok && count > 0 {
			lines = append(lines, "ai overviews "+table.SummaryLine(fieldsOf(overview, "with_ai_overview", "cited", "not_cited", "cited_unknown")))
			if sources := table.AsRows(overview["top_sources"]); len(sources) > 0 {
				lines = append(lines, table.Table(sources, []string{"domain", "keywords"}, nil))
			}
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

func judgmentLines(judgment table.Row) []string {
	if judgment == nil {
		return nil
	}
	var lines []string
	if stance := table.AsObject(judgment["stance"]); len(stance) > 0 {
		lines = append(lines, "stance "+table.SummaryOf(stance))
	}
	if cited := table.AsObject(judgment["rival_cited"]); len(cited) > 0 {
		lines = append(lines, "rivals cited as a source "+table.SummaryOf(cited))
	}
	if unjudged, ok := judgment["cells_unjudged"].(float64); ok && unjudged > 0 {
		lines = append(lines, fmt.Sprintf("unjudged answers %d (regex scoring only)", int(unjudged)))
	}
	if rows := table.AsRows(judgment["review"]); len(rows) > 0 {
		lines = append(lines, "review", table.Table(rows, []string{"prompt", "engine", "subject", "verdict", "confidence"}, nil))
	}
	return lines
}

func accuracyLines(accuracy table.Row) []string {
	if accuracy == nil || accuracy["status"] == "no_facts" {
		return nil
	}
	lines := []string{"accuracy " + table.SummaryLine(fieldsOf(accuracy, "facts", "cells_checked", "cells_unchecked"))}
	if rows := table.AsRows(accuracy["inaccuracies"]); len(rows) > 0 {
		lines = append(lines, "wrong facts", table.Table(rows, []string{"prompt", "engine", "fact", "confidence"}, nil))
	}
	if rows := table.AsRows(accuracy["review"]); len(rows) > 0 {
		lines = append(lines, "facts to review", table.Table(rows, []string{"prompt", "engine", "fact", "confidence"}, nil))
	}
	return lines
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
