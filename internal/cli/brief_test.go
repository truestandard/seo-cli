package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/config"
)

const briefJSON = `{"project":"taffy","cached":false,"cost":0.003,"item":{"id":12,"status":"briefed"},
	"brief":{"job":"create","page_type":"comparison","template":{"name":"comparison"},"keyword":"taffy vs labcorp","reason":"page one is comparisons",
		"intent":{"type":"commercial","first_100_words":"Give the verdict for \"taffy vs labcorp\" in the first lines."},
		"reader":"people who read their own blood tests","title_meta":{"policy":"new","note":"Put the keyword in the title."},
		"cta":{"path":"/switch","label":"Switch to Taffy"},"acceptance":{"pass_score":32,"min_area":3,"grade_max":7,"criteria":[{"id":"a"},{"id":"b"}]},
		"defects":["The profile names no reader."],
		"outline":{"sections":[{"heading":"What an iron panel costs","covered_by":3,"ours":false}]},
		"gaps":[{"label":"Lab history","priority":"medium","covered_by":2}],
		"links":{"inbound":[{"url":"https://taffy.test/guides/ferritin","title":"Ferritin"}]}}}`

func TestBriefLines(t *testing.T) {
	var data map[string]any
	if err := json.Unmarshal([]byte(briefJSON), &data); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(briefLines(data), "\n")
	for _, want := range []string{
		"ticket 12 brief (cost $0.003)", "create comparison page (template comparison): taffy vs labcorp", "why: page one is comparisons",
		"intent commercial: Give the verdict", "reader: people who read", "title and description: new. Put the keyword", "next step: Switch to Taffy (/switch)",
		"pass at 32 of 40, no area under 3, 2 checks, grade 7 or lower", "defect: The profile names no reader.", "What an iron panel costs", "Lab history", "https://taffy.test/guides/ferritin",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	data["cached"] = true
	data["brief"].(map[string]any)["built_on"] = "2026-10-20"
	if out := strings.Join(briefLines(data), "\n"); !strings.Contains(out, "built 2026-10-20, nothing spent") {
		t.Errorf("stored brief line:\n%s", out)
	}
}

func TestRunBriefReadsThenBuildsWhenMissing(t *testing.T) {
	type call struct {
		method string
		body   map[string]any
	}
	var calls []call
	stored := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		payload := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &payload)
		}
		calls = append(calls, call{method: r.Method, body: payload})
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v1/projects/taffy/items/12/brief" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"no route","code":"not_found"}`))
			return
		}
		if r.Method == http.MethodGet && !stored {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"This ticket has no brief yet.","code":"not_found"}`))
			return
		}
		stored = true
		_, _ = w.Write([]byte(briefJSON))
	}))
	defer server.Close()
	client := api.New(config.Config{APIURL: server.URL, Token: "t"})

	if _, err := runBrief(client, "taffy", 12, false, false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].method != http.MethodGet || calls[1].method != http.MethodPost || len(calls[1].body) != 0 {
		t.Fatalf("missing brief should read then build: %+v", calls)
	}

	calls = nil
	if _, err := runBrief(client, "taffy", 12, false, false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet {
		t.Fatalf("stored brief should only read: %+v", calls)
	}

	calls = nil
	if _, err := runBrief(client, "taffy", 12, true, true); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].body["rebuild"] != true || calls[0].body["dry_run"] != true {
		t.Fatalf("--rebuild --dry-run should post both flags: %+v", calls)
	}

	if _, err := runBrief(client, "taffy", 99, false, false); err == nil {
		t.Fatal("an unknown ticket should fail after the build is refused")
	}
}

func TestBriefCommandRejectsABadID(t *testing.T) {
	t.Setenv(config.EnvConfigDir, t.TempDir())
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"--api-url", "http://127.0.0.1:1", "--token", "seo_test", "--project", "taffy", "--quiet", "brief", "abc"})
	err := root.Execute()
	apiErr, ok := err.(*api.APIError)
	if !ok || apiErr.Code != "usage" {
		t.Fatalf("want a usage error, got %v", err)
	}
}
