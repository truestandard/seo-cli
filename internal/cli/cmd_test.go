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
