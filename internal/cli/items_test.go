package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestandard/seo-cli/internal/config"
)

func TestItemsLines(t *testing.T) {
	var listed map[string]any
	raw := `{"open":2,"counts":{"queued":1,"drafting":1,"dropped":3},
		"items":[{"id":7,"kind":"new_page","status":"queued","owner":"autopilot","keyword":"iron panel","target_path":null,"format":"guide","track":"C","last_planned_on":"2026-10-20"},
			{"id":8,"kind":"refresh","status":"drafting","owner":"human","keyword":"ferritin","target_path":"/guides/ferritin"}]}`
	if err := json.Unmarshal([]byte(raw), &listed); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(itemsLines(listed), "\n")
	for _, want := range []string{"2 tickets shown, 2 open", "queued=1", "dropped=3", "iron panel", "/guides/ferritin", "last_planned_on"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	if out := strings.Join(itemsLines(map[string]any{"open": 0, "items": []any{}}), "\n"); !strings.Contains(out, "no tickets match") {
		t.Errorf("empty list:\n%s", out)
	}

	var shown map[string]any
	raw = `{"message":"Ticket 7 is drafting now.","item":{"id":7,"kind":"new_page","status":"drafting","owner":"autopilot","keyword":"iron panel","cluster_key":"iron panel",
		"target_path":"/guides/iron-panel","format":"guide","track":"C","attempts":1,"first_planned_on":"2026-10-13","last_planned_on":"2026-10-20","reason":"900 searches a month",
		"review":{"attempt":1,"verdict":"fail","score":25,"hard_fails":2},"reviews":1,
		"ship_events":[{"url":"https://taffy.test/guides/iron-panel","verified":true,"verified_at":"2026-10-20T10:00:00Z"}],
		"watch":{"url":"https://taffy.test/guides/iron-panel","state":"watching","d7_status":null},
		"children":[{"id":9,"kind":"get_found","status":"queued","cluster_key":"get_found:https://taffy.test/guides/iron-panel"}]}}`
	if err := json.Unmarshal([]byte(raw), &shown); err != nil {
		t.Fatal(err)
	}
	out = strings.Join(itemLines(shown), "\n")
	for _, want := range []string{"Ticket 7 is drafting now.", "#7 new_page drafting (owner autopilot): iron panel", "target=/guides/iron-panel", "planned=2026-10-13..2026-10-20", "900 searches a month",
		"latest check: attempt 1 fail score 25, hard fails 2 (1 checks)", "ships", "watch: https://taffy.test/guides/iron-panel watching (day 7 none", "follow-up tickets", "get_found"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestItemsCommandsCallTheTicketRoutes(t *testing.T) {
	type call struct {
		method string
		path   string
		query  string
		body   map[string]any
	}
	var calls []call
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		payload := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &payload)
		}
		calls = append(calls, call{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: payload})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"open":0,"items":[],"item":{"id":7}}`))
	}))
	defer server.Close()
	t.Setenv(config.EnvConfigDir, t.TempDir())

	runs := [][]string{
		{"items"},
		{"items", "--status", "drafting", "--kind", "refresh", "--limit", "5"},
		{"items", "show", "7"},
		{"items", "update", "7", "--status", "dropped", "--note", "no demand"},
	}
	for _, args := range runs {
		flagDryRun = false
		root := newRootCmd()
		root.SetOut(io.Discard)
		root.SetArgs(append([]string{"--api-url", server.URL, "--token", "seo_test", "--project", "taffy", "--quiet"}, args...))
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	if len(calls) != 4 {
		t.Fatalf("want 4 calls, got %d", len(calls))
	}
	if calls[0].method != http.MethodGet || !strings.HasSuffix(calls[0].path, "/projects/taffy/items") || calls[0].query != "" {
		t.Errorf("plain items should list: %+v", calls[0])
	}
	if calls[1].query != "kind=refresh&limit=5&status=drafting" {
		t.Errorf("filters should pass through: %+v", calls[1])
	}
	if calls[2].method != http.MethodGet || !strings.HasSuffix(calls[2].path, "/projects/taffy/items/7") {
		t.Errorf("show should read one ticket: %+v", calls[2])
	}
	if calls[3].method != http.MethodPatch || !strings.HasSuffix(calls[3].path, "/items/7") || calls[3].body["status"] != "dropped" || calls[3].body["note"] != "no demand" || calls[3].body["owner"] != nil {
		t.Errorf("update should patch the ticket: %+v", calls[3])
	}

	for _, args := range [][]string{{"items", "show", "abc"}, {"items", "update", "7"}} {
		root := newRootCmd()
		root.SetOut(io.Discard)
		root.SetArgs(append([]string{"--api-url", server.URL, "--token", "seo_test", "--project", "taffy", "--quiet"}, args...))
		if err := root.Execute(); err == nil {
			t.Errorf("%v should be a usage error", args)
		}
	}
	if len(calls) != 4 {
		t.Errorf("usage errors must not call the API, got %d calls", len(calls))
	}
}
