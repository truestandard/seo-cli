package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/config"
)

const watchSummaryJSON = `{"total":3,"by_state":{"watching":1,"indexed":0,"found":1,"stalled":1,"closed":0},"by_outcome":{"found":1,"not_indexed":1,"no_impressions":0},
	"open_get_found":1,"last_checked_on":"2026-10-20","stalled":[{"url":"https://x.test/new","outcome":"not_indexed","shipped_on":"2026-10-01","inbound_link":false,"fix":"Link to it from a page Google already shows."}]}`

func TestWatchLinesListsWatches(t *testing.T) {
	var data map[string]any
	raw := `{"project":"x","summary":` + watchSummaryJSON + `,"watches":[{"url":"https://x.test/new","state":"stalled","outcome":"not_indexed","shipped_on":"2026-10-01","d7_status":"not_indexed","d14_status":"not_indexed","d28_impressions":null,"inbound_link":false}]}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := watchLines(data)
	for _, want := range []string{"watched=3", "stalled=1", "open_get_found=1", "last_checked=2026-10-20", "https://x.test/new", "not_indexed"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "stuck pages") {
		t.Errorf("list output repeats the stuck table:\n%s", out)
	}
	empty := watchLines(map[string]any{"summary": map[string]any{"total": 0}, "watches": []any{}})
	if !strings.Contains(empty, "seo watch --backfill") || !strings.Contains(empty, "last_checked=never") {
		t.Errorf("empty state: %s", empty)
	}
}

func TestRunWatchBackfillThenCheck(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/projects/x/watches/backfill":
			_, _ = w.Write([]byte(`{"project":"x","backfill":{"found":4,"created":3,"already_watched":1,"by_source":{"sitemap":2,"backfill":1}},"summary":{"total":3}}`))
		case "/api/v1/projects/x/watches/check":
			_, _ = w.Write([]byte(`{"project":"x","check":{"checked_on":"2026-10-20","checked":3,"newly_indexed":1,"newly_found":1,"newly_stalled":1,"get_found_opened":1,"traffic_window":{"range_start":"2026-09-20","range_end":"2026-10-17"}},"summary":` + watchSummaryJSON + `}`))
		default:
			_, _ = w.Write([]byte(`{"project":"x","summary":{"total":0},"watches":[]}`))
		}
	}))
	defer server.Close()
	client := api.New(config.Config{APIURL: server.URL, Token: "t"})

	data, err := runWatch(client, "x", true, true, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "POST /api/v1/projects/x/watches/backfill?" || calls[1] != "POST /api/v1/projects/x/watches/check?" {
		t.Fatalf("calls: %v", calls)
	}
	out := watchLines(data)
	for _, want := range []string{"backfill: found=4 created=3 already_watched=1 sitemap=2 other=1", "check 2026-10-20", "get_found_opened=1", "2026-09-20..2026-10-17", "watched=3", "stuck pages", "Link to it from a page Google already shows."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	calls = nil
	if _, err := runWatch(client, "x", false, false, "stalled", 20); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "GET /api/v1/projects/x/watches?") || !strings.Contains(calls[0], "state=stalled") || !strings.Contains(calls[0], "limit=20") {
		t.Fatalf("list call: %v", calls)
	}
}
