package cli

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func briefCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "brief ID",
		Short: "show the brief for one ticket, building it when there is none (a build is paid, cents; a stored brief is free)",
		Args:  exactArgs(1, "seo brief ID [--rebuild] [--dry-run]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil || id <= 0 {
				return usageError("usage: seo brief ID [--rebuild] [--dry-run] (ID is the ticket number)")
			}
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := runBrief(client, slug, id, optTrue(cmd, "rebuild") != nil, flagDryRun)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				if table.IsEstimate(data) {
					return table.EstimateLine(data)
				}
				return joinLines(briefLines(table.AsObject(data)), "\n")
			})
			return nil
		},
	}
	c.Flags().Bool("rebuild", false, "build the brief again even when one is stored")
	return c
}

func runBrief(client *api.Client, slug string, id int, rebuild, dryRun bool) (any, error) {
	path := api.ProjectPath(slug, fmt.Sprintf("/items/%d/brief", id))
	build := func() (any, error) {
		payload := map[string]any{}
		if rebuild {
			payload["rebuild"] = true
		}
		if dryRun {
			payload["dry_run"] = true
		}
		return client.Post(path, payload)
	}
	if rebuild || dryRun {
		return build()
	}
	data, err := client.Get(path, nil)
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound && apiErr.Code == "not_found" {
		return build()
	}
	return data, err
}

func briefLines(root table.Row) []string {
	brief := table.AsObject(root["brief"])
	item := table.AsObject(root["item"])
	if brief == nil {
		return []string{"no brief"}
	}
	spent := "cost " + table.Money(root["cost"])
	if root["cached"] == true {
		spent = fmt.Sprintf("built %v, nothing spent", brief["built_on"])
	}
	target := ""
	if path, ok := brief["target_path"].(string); ok && path != "" {
		target = " at " + path
	}
	lines := []string{
		fmt.Sprintf("ticket %v brief (%s)", item["id"], spent),
		fmt.Sprintf("%v %v page (template %v): %v%s", brief["job"], brief["page_type"], table.AsObject(brief["template"])["name"], orDash(brief["keyword"]), target),
	}
	if reason, ok := brief["reason"].(string); ok && reason != "" {
		lines = append(lines, "why: "+reason)
	}
	if intent := table.AsObject(brief["intent"]); intent != nil {
		lines = append(lines, fmt.Sprintf("intent %v: %v", intent["type"], orDash(intent["first_100_words"])))
	}
	lines = append(lines, fmt.Sprintf("reader: %v", orDash(brief["reader"])))
	if title := table.AsObject(brief["title_meta"]); title != nil {
		note := title["risk"]
		if note == nil {
			note = title["note"]
		}
		lines = append(lines, fmt.Sprintf("title and description: %v. %v", title["policy"], orDash(note)))
	}
	if cta := table.AsObject(brief["cta"]); cta != nil {
		lines = append(lines, fmt.Sprintf("next step: %v (%v)", cta["label"], cta["path"]))
	}
	if acceptance := table.AsObject(brief["acceptance"]); acceptance != nil {
		lines = append(lines, fmt.Sprintf("pass at %v of 40, no area under %v, %d checks, grade %v or lower", acceptance["pass_score"], acceptance["min_area"], listLen(acceptance["criteria"]), acceptance["grade_max"]))
	}
	for _, defect := range stringList(brief["defects"]) {
		lines = append(lines, "defect: "+defect)
	}
	if sections := table.AsRows(table.AsObject(brief["outline"])["sections"]); len(sections) > 0 {
		lines = append(lines, "outline from the top pages", table.Table(sections, []string{"heading", "covered_by", "ours"}, nil))
	}
	if gaps := table.AsRows(brief["gaps"]); len(gaps) > 0 {
		lines = append(lines, "gaps to fill", table.Table(gaps, []string{"label", "priority", "covered_by"}, nil))
	}
	if inbound := table.AsRows(table.AsObject(brief["links"])["inbound"]); len(inbound) > 0 {
		lines = append(lines, "pages that should link here", table.Table(inbound, []string{"url", "title"}, nil))
	}
	return lines
}

func orDash(value any) any {
	if value == nil || value == "" {
		return "-"
	}
	return value
}

func stringList(value any) []string {
	list, _ := value.([]any)
	out := make([]string, 0, len(list))
	for _, entry := range list {
		if text, ok := entry.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}
