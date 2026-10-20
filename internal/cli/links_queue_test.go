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

func TestLinkQueueLines(t *testing.T) {
	var built map[string]any
	raw := `{"url":"https://taffy.test/guides/iron","cost":0.0021,"pause":{"paused":true,"reason":"New pages wait. You have 13 linking sites and page one needs about 40."},
		"link_queue":{"built_on":"2026-10-20","cached":false,"opened_items":2,
			"internal":[{"url":"https://taffy.test/guides/ferritin","title":"Ferritin","links_now":false},{"url":"https://taffy.test/","title":"Home","links_now":true}],
			"external":[{"domain":"roundup.test","reason":"roundup.test links to alpha.test and beta.test. Ask it to link to you too."}]}}`
	if err := json.Unmarshal([]byte(raw), &built); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(linkQueueLines(built), "\n")
	for _, want := range []string{"New pages wait", "https://taffy.test/guides/iron (cost $0.0021, 2 link items opened)", "link from your pages", "https://taffy.test/guides/ferritin", "outside sites to ask", "roundup.test"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	var listed map[string]any
	raw = `{"built":1,"due":2,"open_link_items":3,"pause":{"reason":"New pages can go out."},
		"watches":[{"url":"https://taffy.test/guides/iron","state":"watching","link_queue":{"built_on":"2026-10-20","internal":[{"links_now":false},{"links_now":true}],"external":[{"domain":"a.test"}]}},
			{"url":"https://taffy.test/guides/new","state":"watching","link_queue":null}]}`
	if err := json.Unmarshal([]byte(raw), &listed); err != nil {
		t.Fatal(err)
	}
	out = strings.Join(linkQueueLines(listed), "\n")
	for _, want := range []string{"New pages can go out.", "1 pages have link plans, 2 due; open link items: 3", "https://taffy.test/guides/new", "built_on"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	var run map[string]any
	raw = `{"due":3,"cost":0.004,"built":[{"url":"https://taffy.test/a","internal":2,"external":1,"opened_items":2}],"skipped_for_budget":["https://taffy.test/b"],"failed":[]}`
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		t.Fatal(err)
	}
	out = strings.Join(linkQueueLines(run), "\n")
	if !strings.Contains(out, "built 1, skipped at the cap 1, failed 0 (cost $0.004)") || !strings.Contains(out, "https://taffy.test/a") {
		t.Errorf("run summary:\n%s", out)
	}
}

func TestLinksQueueCommandReadsOrBuilds(t *testing.T) {
	type call struct {
		method string
		path   string
		body   map[string]any
	}
	var calls []call
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		payload := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &payload)
		}
		calls = append(calls, call{method: r.Method, path: r.URL.Path, body: payload})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"built":0,"due":0,"watches":[]}`))
	}))
	defer server.Close()
	t.Setenv(config.EnvConfigDir, t.TempDir())

	runs := [][]string{
		{"links", "queue"},
		{"links", "queue", "--url", "https://taffy.test/guides/iron", "--force"},
		{"links", "queue", "--dry-run"},
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

	if len(calls) != 3 {
		t.Fatalf("want 3 calls, got %d", len(calls))
	}
	if calls[0].method != http.MethodGet || !strings.HasSuffix(calls[0].path, "/projects/taffy/link_queue") {
		t.Errorf("plain queue should read: %+v", calls[0])
	}
	if calls[1].method != http.MethodPost || calls[1].body["url"] != "https://taffy.test/guides/iron" || calls[1].body["force"] != true {
		t.Errorf("--url should build that page: %+v", calls[1])
	}
	if calls[2].method != http.MethodPost || calls[2].body["dry_run"] != true || calls[2].body["url"] != nil {
		t.Errorf("--dry-run should price every due page: %+v", calls[2])
	}
}
