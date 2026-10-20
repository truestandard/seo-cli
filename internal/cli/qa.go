package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

var qaExit = os.Exit

func qaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "qa",
		Short: "check a draft against the brief and the product profile (Jev, cents); exit 1 on a fail, 10 when a person must read it",
		Args:  exactArgs(0, "seo qa --item ID --file PATH [--format html] [--adapters results.json] [--url URL] [--dry-run]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			file, _ := cmd.Flags().GetString("file")
			if file == "" {
				return usageError("usage: seo qa --item ID --file PATH [--format html] [--adapters results.json] [--dry-run]")
			}
			payload, err := qaPayload(cmd, file)
			if err != nil {
				return err
			}
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Post(api.ProjectPath(slug, "/reviews"), body(payload))
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return joinLines(qaLines(table.AsObject(data)), "\n")
			})
			if code := qaExitCode(data); code != output.ExitOK {
				qaExit(code)
			}
			return nil
		},
	}
	c.Flags().String("item", "", "plan item id; the check counts as one of its rounds")
	c.Flags().String("file", "", "the draft to check")
	c.Flags().String("format", "", "markdown or html (default: html for .html and .htm files, else markdown)")
	c.Flags().String("adapters", "", "JSON file with the product's check adapter results: [{name, hard_fail, passed or exit_code}]")
	c.Flags().String("url", "", "where the page will live")
	return c
}

func qaPayload(cmd *cobra.Command, file string) (map[string]any, error) {
	draft, err := os.ReadFile(file)
	if err != nil {
		return nil, usageError(err.Error())
	}
	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = "markdown"
		if ext := strings.ToLower(filepath.Ext(file)); ext == ".html" || ext == ".htm" {
			format = "html"
		}
	}
	if format != "markdown" && format != "html" {
		return nil, usageError("--format must be markdown or html")
	}
	payload := map[string]any{"draft": string(draft), "draft_format": format, "url": optString(cmd, "url")}
	if item := optString(cmd, "item"); item != nil {
		id, err := strconv.Atoi(strings.TrimSpace(item.(string)))
		if err != nil {
			return nil, usageError("--item must be a plan item id")
		}
		payload["plan_item_id"] = id
	}
	if path, _ := cmd.Flags().GetString("adapters"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, usageError(err.Error())
		}
		var results any
		if err := json.Unmarshal(raw, &results); err != nil {
			return nil, usageError("--adapters is not valid JSON: " + err.Error())
		}
		payload["adapter_results"] = results
	}
	return payload, nil
}

func qaExitCode(data any) int {
	if table.IsEstimate(data) {
		return output.ExitOK
	}
	switch table.AsObject(data)["verdict"] {
	case "pass":
		return output.ExitOK
	case "needs_human":
		return output.ExitNeedsClarification
	default:
		return output.ExitError
	}
}

func qaLines(root table.Row) []string {
	lines := []string{fmt.Sprintf("review %v: %v, score %v of 40 (cost %s)", root["id"], root["verdict"], orZero(root["score"]), table.Money(root["cost"]))}
	if item := table.AsObject(root["plan_item"]); item != nil {
		lines = append(lines, fmt.Sprintf("item %v is %v after %v of %v rounds", item["id"], item["status"], item["attempts"], item["max_attempts"]))
	}
	if reasons, ok := root["reasons"].([]any); ok {
		for _, reason := range reasons {
			lines = append(lines, fmt.Sprint(reason))
		}
	}
	if areas := table.AsObject(root["areas"]); areas != nil {
		parts := []string{}
		for _, area := range []string{"intent", "structure", "voice", "seo", "product_accuracy", "sources", "links_assets_schema", "publish_ready"} {
			if value, ok := areas[area]; ok {
				parts = append(parts, fmt.Sprintf("%s %v", area, value))
			}
		}
		lines = append(lines, "areas: "+strings.Join(parts, ", "))
	}
	if hard := table.AsRows(root["hard_fails"]); len(hard) > 0 {
		lines = append(lines, "hard fails", table.Table(hard, []string{"id", "paragraph", "evidence"}, nil))
	}
	if fixes := table.AsRows(root["fixes"]); len(fixes) > 0 {
		lines = append(lines, "fix only these", table.Table(fixes, []string{"paragraph", "criterion", "evidence", "how"}, nil))
	}
	return lines
}
