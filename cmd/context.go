package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func contextCmd() *cobra.Command {
	c := &cobra.Command{Use: "context", Short: "project memory: sections, competitors, key pages, research log"}
	get := contextGetCmd()
	c.RunE = get.RunE
	c.Args = exactArgs(0, "seo context [get | set | add-competitor | add-page | log]")
	c.AddCommand(get, contextSetCmd(), contextAddCompetitorCmd(), contextAddPageCmd(), contextLogCmd())
	return c
}

func contextGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "print the project context",
		Args:  exactArgs(0, "seo context get"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/context"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return renderContext(data) })
			return nil
		},
	}
}

func contextPatch(patch map[string]any, human string) error {
	client, slug, err := projectClient()
	if err != nil {
		return err
	}
	data, err := client.Patch(api.ProjectPath(slug, "/context"), patch)
	if err != nil {
		return err
	}
	output.Render(data, func() string { return human })
	return nil
}

func contextSetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "set <key> [text...]",
		Short: "set a context section (reads stdin when text is omitted)",
		Args:  minArgs(1, "seo context set <key> [text...] [--file <path>]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			content, err := readText(args[1:], file)
			if err != nil {
				return err
			}
			return contextPatch(map[string]any{"sections": map[string]any{args[0]: content}},
				fmt.Sprintf("set section %s (%d chars)", args[0], len(content)))
		},
	}
	c.Flags().String("file", "", "read the section content from a file")
	return c
}

func contextAddCompetitorCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add-competitor <domain>",
		Short: "add a competitor domain",
		Args:  exactArgs(1, "seo context add-competitor <domain> [--name] [--notes]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			competitor := map[string]any{"domain": args[0]}
			if v := optString(cmd, "name"); v != nil {
				competitor["name"] = v
			}
			if v := optString(cmd, "notes"); v != nil {
				competitor["notes"] = v
			}
			return contextPatch(map[string]any{"add_competitors": []any{competitor}}, "added competitor "+args[0])
		},
	}
	c.Flags().String("name", "", "competitor name")
	c.Flags().String("notes", "", "notes")
	return c
}

func contextAddPageCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add-page <path>",
		Short: "add a key page",
		Args:  exactArgs(1, "seo context add-page <path> [--role] [--topic]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			page := map[string]any{"path": args[0]}
			if v := optString(cmd, "role"); v != nil {
				page["role"] = v
			}
			if v := optString(cmd, "topic"); v != nil {
				page["topic"] = v
			}
			return contextPatch(map[string]any{"add_key_pages": []any{page}}, "added key page "+args[0])
		},
	}
	c.Flags().String("role", "", "page role, e.g. landing, pricing, blog")
	c.Flags().String("topic", "", "topic the page targets")
	return c
}

func contextLogCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "log <kind> <summary...>",
		Short: "append a research log entry",
		Args:  minArgs(2, "seo context log <kind> <summary...> [--inputs <json>]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			entry := map[string]any{"kind": args[0], "summary": joinArgs(args[1:])}
			if raw, _ := cmd.Flags().GetString("inputs"); raw != "" {
				var inputs map[string]any
				if err := json.Unmarshal([]byte(raw), &inputs); err != nil || inputs == nil {
					return usageError("--inputs must be a JSON object")
				}
				entry["inputs"] = inputs
			}
			return contextPatch(map[string]any{"research_log": entry}, "logged "+args[0])
		},
	}
	c.Flags().String("inputs", "", "JSON object describing the inputs")
	return c
}

func renderContext(data any) string {
	root := table.First(table.AsObject(data), "context")
	var parts []string
	switch sections := root["sections"].(type) {
	case []any:
		for _, section := range table.AsRows(sections) {
			parts = append(parts, "## "+table.Cell(section["key"])+"\n"+table.Cell(section["content"]))
		}
	case map[string]any:
		for _, pair := range pairsOf(sections) {
			content := pair.Value
			if nested := table.AsObject(content); nested != nil && nested["content"] != nil {
				content = nested["content"]
			}
			parts = append(parts, "## "+pair.Key+"\n"+table.Cell(content))
		}
	}
	if rows := table.AsRows(root["competitors"]); len(rows) > 0 {
		parts = append(parts, "## competitors\n"+table.Table(rows, []string{"domain", "name", "notes"}, nil))
	}
	if rows := table.AsRows(root["key_pages"]); len(rows) > 0 {
		parts = append(parts, "## key pages\n"+table.Table(rows, []string{"path", "role", "topic"}, nil))
	}
	if rows := table.AsRows(root["research_log"]); len(rows) > 0 {
		parts = append(parts, "## research log\n"+table.Table(rows, []string{"created_at", "kind", "summary"}, nil))
	}
	if len(parts) == 0 {
		remaining := table.Row{}
		for key, value := range root {
			switch key {
			case "sections", "competitors", "key_pages", "research_log":
			default:
				remaining[key] = value
			}
		}
		if len(remaining) == 0 {
			return "context is empty"
		}
		return table.KeyValueBlock(pairsOf(remaining))
	}
	return joinLines(parts, "\n\n")
}
