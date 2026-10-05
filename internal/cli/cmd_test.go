package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSince(t *testing.T) {
	for _, ok := range []string{"7d", "30d", "2026-08-01"} {
		if got, err := since(ok); err != nil || got != ok {
			t.Errorf("%s: %v %q", ok, err, got)
		}
	}
	for _, bad := range []string{"7", "2026-13-01", "last week"} {
		if _, err := since(bad); err == nil {
			t.Errorf("%s should fail", bad)
		}
	}
}

func TestPaidList(t *testing.T) {
	if !IsPaid("track") || IsPaid("ranks") || IsPaid("scoreboard") {
		t.Fatal("paid list wrong")
	}
}

func TestWithDefaultProject(t *testing.T) {
	schema := map[string]any{"properties": map[string]any{"project": map[string]any{}, "since": map[string]any{}}}
	args := WithDefaultProject(schema, json.RawMessage(`{"since":"7d"}`), "precis")
	if args["project"] != "precis" || args["since"] != "7d" {
		t.Fatalf("got %v", args)
	}
	args = WithDefaultProject(schema, json.RawMessage(`{"project":"other"}`), "precis")
	if args["project"] != "other" {
		t.Fatalf("explicit project overwritten: %v", args)
	}
	args = WithDefaultProject(map[string]any{"properties": map[string]any{"q": nil}}, nil, "precis")
	if _, has := args["project"]; has {
		t.Fatal("project injected into a tool that does not take one")
	}
}

func TestBodyAddsDryRunOnlyWhenSet(t *testing.T) {
	flagDryRun = false
	if _, has := body(map[string]any{"a": nil})["dry_run"]; has {
		t.Fatal("dry_run leaked")
	}
	flagDryRun = true
	defer func() { flagDryRun = false }()
	payload := body(map[string]any{"a": nil, "b": 1})
	if payload["dry_run"] != true || payload["b"] != 1 {
		t.Fatalf("got %v", payload)
	}
	if _, has := payload["a"]; has {
		t.Fatal("nil field kept")
	}
}

func TestRenderVisibilityShowsJudgment(t *testing.T) {
	var data any
	raw := `{"report":{"cells":6,"competitor_share_of_voice":{"Examine":2},"competitor_share_of_voice_method":"jev_suggested",
		"judgment":{"stance":{"one_of_several":1},"rival_cited":{"WebMD":1},"cells_unjudged":2,"review":[{"prompt":"won query","engine":"gpt-5.5","subject":"Healthline","verdict":"incidental","confidence":0.45}]}}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderVisibility(data)
	for _, want := range []string{"rivals recommended", "Examine", "stance", "one_of_several", "unjudged answers 2", "Healthline", "incidental", "cited as a source WebMD"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestLinkSourceLines(t *testing.T) {
	var data map[string]any
	raw := `{"status":"ranked","method":"jev","candidates":12,"sources":[{"path":"/guides/iron","title":"Iron","level":3.5,"links_now":true}],"already_linking":["https://x.test/a"]}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(linkSourceLines(data, []string{"path", "title", "level", "links_now"}), "\n")
	for _, want := range []string{"by jev over 12", "/guides/iron", "already linking: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if got := linkSourceLines(map[string]any{"status": "no_audit"}, nil); !strings.Contains(got[0], "no completed site audit") {
		t.Errorf("no_audit line: %v", got)
	}
}

func TestTriageLines(t *testing.T) {
	var data map[string]any
	raw := `{"status":"triaged","counts":{"buyer":2,"job_seeker":1},"warnings":["no context"],
		"groups":[{"head":"ai audit","volume":199,"keywords":["ai audit","audit ai"]}],
		"keywords":[{"keyword":"ai consultant salary","relevance":"job_seeker","confidence":0.99,"keep":false},{"keyword":"ai audit","relevance":"buyer","keep":true}],
		"review":{"relevance":[{"keyword":"x"}],"pairs":[]}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(triageLines(data), "\n")
	for _, want := range []string{"buyer", "warning: no context", "ai audit, audit ai", "screened out", "ai consultant salary", "review: 1 unclear"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestPlanLines(t *testing.T) {
	var data map[string]any
	raw := `{"status":"planned","cost":0.02,
		"fixes":[{"severity":"critical","rule":"sitemap_non_200","count":2,"new":1,"how_to_fix":"Fix it."}],
		"refreshes":[{"source":"decay","keyword":"comment search","page":"/cs","position":12,"value":4.1}],
		"new_pages":{"articles":[{"keyword":"how to search youtube comments","track":"C","format":"guide","floor":"winnable","expected_clicks":72,"drafting":"serp-outline + ai-seo"}],"comparisons":[],"free_tools":[{"keyword":"youtube comment picker","track":"tool","format":"tool"}]},
		"needs_human":[{"kind":"forum_reply","keyword":"old comments","human_reason":"forum threads hold page one"}],
		"deferred":[{"keyword":"x"}]}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := planLines(data)
	for _, want := range []string{"site fixes", "sitemap_non_200", "refreshes", "decay", "articles", "free_tools", "youtube comment picker", "needs a person", "forum threads", "deferred: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if !IsPaid("plan") {
		t.Error("plan should be paid")
	}
}

func TestParseSettings(t *testing.T) {
	settings, err := parseSettings([]string{"articles_per_month=12", "plan_spend_cap=2.5", "site_watch=false", "ship_mode=pr"})
	if err != nil {
		t.Fatal(err)
	}
	if settings["articles_per_month"] != 12 || settings["plan_spend_cap"] != 2.5 || settings["site_watch"] != false || settings["ship_mode"] != "pr" {
		t.Errorf("parsed %v", settings)
	}
	if _, err := parseSettings([]string{"nokey"}); err == nil {
		t.Error("want a usage error")
	}
}

func TestRenderFixes(t *testing.T) {
	var data any
	raw := `{"status":"ready","scan_id":7,"measured_at":"2026-10-04","actions":[
		{"kind":"correct_fact","priority":1,"prompt":"what does taffy cost","page":null,"engines":["perplexity-sonar-pro"],"needs_human":true},
		{"kind":"aio_citable","priority":2,"keyword":"youtube comment search","page":"/","engines":["google-ai-overview"],"needs_human":false}]}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderFixes(data)
	for _, want := range []string{"ai fixes: ready, scan 7", "2 actions, 1 need a person", "correct_fact", "perplexity-sonar-pro", "youtube comment search", "person"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if got := renderFixes(map[string]any{"status": "no_scan", "actions": []any{}}); !strings.Contains(got, "no actions") {
		t.Errorf("empty fixes: %s", got)
	}
}

func TestRenderVisibilityShowsAccuracyAndOverviews(t *testing.T) {
	var data any
	raw := `{"report":{"cells":8,"accuracy":{"status":"checked","facts":3,"cells_checked":4,"cells_unchecked":1,
		"inaccuracies":[{"prompt":"taffy pricing","engine":"gpt-5.6-sol","fact":"Pro costs $19 a month","confidence":0.91}],"review":[]}},
		"supplementary":{"report":{"models":["perplexity/sonar-pro"],"cells":4,"named":1,"cited":0,"queries_hit":1}},
		"ai_overview":{"with_ai_overview":3,"cited":0,"not_cited":3,"cited_unknown":0,"top_sources":[{"domain":"reddit.com","keywords":2}]}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderVisibility(data)
	for _, want := range []string{"accuracy", "cells_unchecked", "wrong facts", "Pro costs $19 a month", "supplementary", "perplexity/sonar-pro", "ai overviews", "reddit.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCompetitorWatchLines(t *testing.T) {
	var data map[string]any
	raw := `{"status":"ok","taken_on":"2026-10-07","cost":"0.1455","competitors":[{"domain":"vidiq.com","baseline":false,"removed_pages_count":2,
		"new_pages":[{"url":"https://vidiq.com/blog/comment-search","title":"Search comments","relevance":"competes_for_our_buyers"},{"url":"https://vidiq.com/careers","title":null,"relevance":"irrelevant"}],
		"keywords_gained":[{"keyword":"youtube comment search","position":7}],"keywords_lost":[],"error":null}],
		"opportunities":[{"keyword":"youtube comment search","volume":1200,"kd":14,"competitor":"vidiq.com","their_position":7,"relevance":"buyer","tracked":false,"our_position":null}],
		"run":{"spent":"0.1455","cap":"1.0"}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(competitorWatchLines(data), "\n")
	for _, want := range []string{"competitor watch ok on 2026-10-07", "vidiq.com", "compete for your buyers", "Search comments", "opportunities", "youtube comment search", "spent $0.1455 of a $1.0 cap"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "https://vidiq.com/careers") {
		t.Errorf("irrelevant page listed:\n%s", out)
	}
	if lines := competitorWatchLines(map[string]any{"status": "none"}); !strings.Contains(lines[0], "--run") {
		t.Errorf("empty watch should point at --run: %v", lines)
	}
}

func TestRenderWatch(t *testing.T) {
	var data any
	raw := `{"run_id":12,"finished_at":"2026-10-04T06:01:00Z","status":"completed",
		"new_issues":[{"severity":"critical","rule":"broken_internal_link","url":"https://x.test/old","detail":"returned HTTP 404"}],
		"resolved":[{"severity":"warning","rule":"orphan_page","url":"https://x.test/a"}],
		"open_critical":[{"rule":"broken_internal_link","url":"https://x.test/old","how_to_fix":"Fix the page"}],
		"open_warning_count":3,
		"rule_counts":{"broken_internal_link":{"before":0,"after":1},"missing_h1":{"before":2,"after":2}}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderWatch(data)
	for _, want := range []string{"run=12", "new=1", "resolved=1", "open_warnings=3", "broken_internal_link", "https://x.test/old", "orphan_page", "rule counts that moved"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "missing_h1") {
		t.Errorf("unchanged rule listed:\n%s", out)
	}
	if got := renderWatch(map[string]any{"status": "no_run"}); !strings.Contains(got, "no completed site audit") {
		t.Errorf("no_run: %q", got)
	}
}
