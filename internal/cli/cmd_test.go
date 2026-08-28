package cli

import (
	"encoding/json"
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
