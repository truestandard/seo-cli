package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func itemsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "items",
		Short: "tickets: every planned page job with its status (free)",
		Args:  exactArgs(0, "seo items [--status S] [--kind K] [--limit N] | seo items show ID | seo items update ID --status S"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			query := api.Query{"status": flagString(cmd, "status"), "kind": flagString(cmd, "kind")}
			if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
				query["limit"] = strconv.Itoa(limit)
			}
			data, err := client.Get(api.ProjectPath(slug, "/items"), query)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return joinLines(itemsLines(table.AsObject(data)), "\n") })
			return nil
		},
	}
	c.Flags().String("status", "", "open (default), closed, all, or one status: queued, briefed, drafting, in_review, approved, shipped, watching, done, failed, needs_human, dropped")
	c.Flags().String("kind", "", "new_page, rewrite_existing, refresh, merge, fix, get_found or link")
	c.Flags().Int("limit", 0, "rows to list (default 100, max 1000)")
	c.AddCommand(itemsShowCmd(), itemsUpdateCmd())
	return c
}

func itemsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show ID",
		Short: "one ticket with its brief, latest check, ships and page watch (free)",
		Args:  exactArgs(1, "seo items show ID"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			id, err := itemID(args[0])
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/items/"+id), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return joinLines(itemLines(table.AsObject(data)), "\n") })
			return nil
		},
	}
}

func itemsUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update ID",
		Short: "move a ticket by hand: queued or briefed to drafting, any open to needs_human or dropped (with --note), needs_human to queued",
		Args:  exactArgs(1, "seo items update ID [--status S] [--owner O] [--note N]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			id, err := itemID(args[0])
			if err != nil {
				return err
			}
			fields := map[string]any{"status": optString(cmd, "status"), "owner": optString(cmd, "owner"), "note": optString(cmd, "note")}
			if fields["status"] == nil && fields["owner"] == nil && fields["note"] == nil {
				return usageError("usage: seo items update ID --status S [--owner O] [--note N]")
			}
			data, err := client.Patch(api.ProjectPath(slug, "/items/"+id), body(fields))
			if err != nil {
				return err
			}
			output.Render(data, func() string { return joinLines(itemLines(table.AsObject(data)), "\n") })
			return nil
		},
	}
	c.Flags().String("status", "", "new status: drafting, needs_human, dropped or queued")
	c.Flags().String("owner", "", "autopilot or human")
	c.Flags().String("note", "", "why; required to drop a ticket")
	return c
}

func itemID(raw string) (string, error) {
	if _, err := strconv.Atoi(raw); err != nil {
		return "", usageError(fmt.Sprintf("ticket id must be a number, got %q", raw))
	}
	return raw, nil
}

func itemsLines(root table.Row) []string {
	items := table.AsRows(root["items"])
	lines := []string{fmt.Sprintf("%d tickets shown, %v open", len(items), orZero(root["open"]))}
	if counts := table.AsObject(root["counts"]); len(counts) > 0 {
		pairs := []table.Pair{}
		for _, status := range []string{"queued", "briefed", "drafting", "in_review", "approved", "shipped", "watching", "needs_human", "done", "failed", "dropped"} {
			if value, ok := counts[status]; ok {
				pairs = append(pairs, table.Pair{Key: status, Value: value})
			}
		}
		lines = append(lines, table.SummaryLine(pairs))
	}
	if len(items) > 0 {
		lines = append(lines, table.Table(items, []string{"id", "kind", "status", "owner", "keyword", "target_path", "format", "track", "last_planned_on"}, nil))
	} else {
		lines = append(lines, "no tickets match: seo plan opens them")
	}
	return lines
}

func itemLines(root table.Row) []string {
	item := table.AsObject(root["item"])
	lines := []string{}
	if message, ok := root["message"].(string); ok && message != "" {
		lines = append(lines, message)
	}
	title := item["keyword"]
	if title == nil {
		title = item["cluster_key"]
	}
	lines = append(lines, fmt.Sprintf("#%v %v %v (owner %v): %v", item["id"], item["kind"], item["status"], orNone(item["owner"]), title))
	lines = append(lines, table.SummaryLine([]table.Pair{
		{Key: "target", Value: orNone(item["target_path"])}, {Key: "format", Value: orNone(item["format"])},
		{Key: "track", Value: orNone(item["track"])}, {Key: "attempts", Value: orZero(item["attempts"])},
		{Key: "planned", Value: fmt.Sprintf("%v..%v", orNone(item["first_planned_on"]), orNone(item["last_planned_on"]))},
	}))
	if reason, ok := item["reason"].(string); ok && reason != "" {
		lines = append(lines, reason)
	}
	if parent := table.AsObject(item["parent"]); parent != nil {
		lines = append(lines, fmt.Sprintf("parent #%v %v %v", parent["id"], parent["kind"], parent["status"]))
	}
	if review := table.AsObject(item["review"]); review != nil {
		lines = append(lines, fmt.Sprintf("latest check: attempt %v %v score %v, hard fails %v (%v checks)", review["attempt"], review["verdict"], orNone(review["score"]), orZero(review["hard_fails"]), orZero(item["reviews"])))
	}
	if ships := table.AsRows(item["ship_events"]); len(ships) > 0 {
		lines = append(lines, "ships", table.Table(ships, []string{"url", "verified", "verified_at"}, nil))
	}
	if watch := table.AsObject(item["watch"]); watch != nil {
		lines = append(lines, fmt.Sprintf("watch: %v %v (day 7 %v, day 14 %v, day 28 views %v)", watch["url"], watch["state"], orNone(watch["d7_status"]), orNone(watch["d14_status"]), orNone(watch["d28_impressions"])))
	}
	if children := table.AsRows(item["children"]); len(children) > 0 {
		lines = append(lines, "follow-up tickets", table.Table(children, []string{"id", "kind", "status", "cluster_key"}, nil))
	}
	return lines
}

func orNone(value any) any {
	if value == nil {
		return "none"
	}
	return value
}
