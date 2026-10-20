package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

var watchStates = []string{"watching", "indexed", "found", "stalled", "closed"}

func watchCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "watch",
		Short: "watch every live page: index on day 7 and 14, impressions on day 28",
		Args:  maxArgs(0, "seo watch [--state S] [--limit N] [--backfill] [--check]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			backfill, _ := cmd.Flags().GetBool("backfill")
			check, _ := cmd.Flags().GetBool("check")
			limit, _ := cmd.Flags().GetInt("limit")
			data, err := runWatch(client, slug, backfill, check, flagString(cmd, "state"), limit)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return watchLines(data) })
			return nil
		},
	}
	c.Flags().String("state", "", "only watches in this state: watching, indexed, found, stalled or closed")
	c.Flags().Int("limit", 0, "rows to list (default 100, max 1000)")
	c.Flags().Bool("backfill", false, "start a watch for every live page the app already knows (free)")
	c.Flags().Bool("check", false, "run today's day 7, 14 and 28 checkpoints and open get_found tickets for misses (free)")
	return c
}

func runWatch(client *api.Client, slug string, backfill, check bool, state string, limit int) (any, error) {
	if !backfill && !check {
		query := api.Query{"state": state}
		if limit > 0 {
			query["limit"] = strconv.Itoa(limit)
		}
		return client.Get(api.ProjectPath(slug, "/watches"), query)
	}
	merged := table.Row{"project": slug}
	if backfill {
		data, err := client.Post(api.ProjectPath(slug, "/watches/backfill"), nil)
		if err != nil {
			return nil, err
		}
		root := table.AsObject(data)
		merged["backfill"], merged["summary"] = root["backfill"], root["summary"]
	}
	if check {
		data, err := client.Post(api.ProjectPath(slug, "/watches/check"), nil)
		if err != nil {
			return nil, err
		}
		root := table.AsObject(data)
		merged["check"], merged["summary"] = root["check"], root["summary"]
	}
	return merged, nil
}

func watchLines(data any) string {
	root := table.AsObject(data)
	summary := table.AsObject(root["summary"])
	var lines []string
	if backfill := table.AsObject(root["backfill"]); backfill != nil {
		sources := table.AsObject(backfill["by_source"])
		lines = append(lines, "backfill: "+table.SummaryLine([]table.Pair{
			{Key: "found", Value: backfill["found"]}, {Key: "created", Value: backfill["created"]},
			{Key: "already_watched", Value: backfill["already_watched"]},
			{Key: "sitemap", Value: sources["sitemap"]}, {Key: "other", Value: sources["backfill"]},
		}))
	}
	if check := table.AsObject(root["check"]); check != nil {
		window := "none"
		if traffic := table.AsObject(check["traffic_window"]); traffic != nil {
			window = fmt.Sprintf("%v..%v", traffic["range_start"], traffic["range_end"])
		}
		lines = append(lines, fmt.Sprintf("check %v: ", check["checked_on"])+table.SummaryLine([]table.Pair{
			{Key: "checked", Value: check["checked"]}, {Key: "newly_indexed", Value: check["newly_indexed"]},
			{Key: "newly_found", Value: check["newly_found"]}, {Key: "newly_stalled", Value: check["newly_stalled"]},
			{Key: "get_found_opened", Value: check["get_found_opened"]}, {Key: "search_console_window", Value: window},
		}))
	}
	if summary != nil {
		lines = append(lines, watchSummaryLine(summary))
	}
	if watches := table.AsRows(root["watches"]); len(watches) > 0 {
		lines = append(lines, table.Table(watches, []string{"url", "state", "outcome", "shipped_on", "d7_status", "d14_status", "d28_impressions", "d28_clicks", "inbound_link"}, nil))
	} else if root["watches"] != nil {
		lines = append(lines, "no watches yet: seo watch --backfill")
	}
	if stalled := table.AsRows(summary["stalled"]); len(stalled) > 0 && root["watches"] == nil {
		lines = append(lines, "stuck pages", table.Table(stalled, []string{"url", "outcome", "shipped_on", "inbound_link", "fix"}, nil))
	}
	return joinLines(lines, "\n")
}

func watchSummaryLine(summary table.Row) string {
	states := table.AsObject(summary["by_state"])
	pairs := []table.Pair{{Key: "watched", Value: summary["total"]}}
	for _, state := range watchStates {
		pairs = append(pairs, table.Pair{Key: state, Value: states[state]})
	}
	lastChecked := summary["last_checked_on"]
	if lastChecked == nil {
		lastChecked = "never"
	}
	pairs = append(pairs, table.Pair{Key: "open_get_found", Value: summary["open_get_found"]}, table.Pair{Key: "last_checked", Value: lastChecked})
	return table.SummaryLine(pairs)
}
