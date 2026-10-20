package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestandard/seo-cli/internal/config"
	"github.com/truestandard/seo-cli/internal/output"
)

func TestQaLines(t *testing.T) {
	var review map[string]any
	raw := `{"id":12,"verdict":"fail","score":27,"cost":0.0004,"reasons":["2 hard fails: banned_punctuation, plain_words."],
		"areas":{"intent":4,"structure":5,"voice":1,"seo":5,"product_accuracy":5,"sources":5,"links_assets_schema":3,"publish_ready":5},
		"plan_item":{"id":7,"status":"drafting","attempts":1,"max_attempts":3},
		"hard_fails":[{"id":"banned_punctuation","paragraph":2,"evidence":"\"—\": Iron matters — a lot."}],
		"fixes":[{"paragraph":2,"criterion":"banned_punctuation","evidence":"Iron matters","how":"Remove the dash and use a period or a comma."}]}`
	if err := json.Unmarshal([]byte(raw), &review); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(qaLines(review), "\n")
	for _, want := range []string{"review 12: fail, score 27 of 40 (cost $0.0004)", "item 7 is drafting after 1 of 3 rounds", "2 hard fails", "voice 1", "hard fails", "fix only these", "Remove the dash"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestQaExitCode(t *testing.T) {
	cases := map[string]int{
		`{"verdict":"pass"}`:                 output.ExitOK,
		`{"verdict":"fail"}`:                 output.ExitError,
		`{"verdict":"needs_human"}`:          output.ExitNeedsClarification,
		`{"estimate":{"cost":0.0002}}`:       output.ExitOK,
		`{"verdict":"pending","score":null}`: output.ExitError,
	}
	for raw, want := range cases {
		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			t.Fatal(err)
		}
		if got := qaExitCode(data); got != want {
			t.Errorf("%s: want exit %d, got %d", raw, want, got)
		}
	}
}

func TestQaCommandPostsTheDraftAndExitsOnFail(t *testing.T) {
	var bodies []map[string]any
	var paths []string
	reply := `{"id":3,"verdict":"fail","score":20,"hard_fails":[],"fixes":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		payload := map[string]any{}
		_ = json.Unmarshal(raw, &payload)
		bodies = append(bodies, payload)
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if payload["dry_run"] == true {
			_, _ = w.Write([]byte(`{"estimate":{"cost":0.0002,"calls":2}}`))
			return
		}
		_, _ = w.Write([]byte(reply))
	}))
	defer server.Close()
	t.Setenv(config.EnvConfigDir, t.TempDir())

	dir := t.TempDir()
	draft := filepath.Join(dir, "draft.html")
	if err := os.WriteFile(draft, []byte("<h1>Iron</h1><p>Iron helps.</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapters := filepath.Join(dir, "adapters.json")
	if err := os.WriteFile(adapters, []byte(`[{"name":"numbers_check","hard_fail":true,"exit_code":1}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	var exits []int
	qaExit = func(code int) { exits = append(exits, code) }
	defer func() { qaExit = os.Exit }()

	runs := [][]string{
		{"qa", "--item", "7", "--file", draft, "--adapters", adapters},
		{"qa", "--file", draft, "--format", "markdown", "--dry-run"},
	}
	for _, args := range runs {
		flagDryRun = false
		root := newRootCmd()
		root.SetOut(io.Discard)
		output.Stdout = io.Discard
		root.SetArgs(append([]string{"--api-url", server.URL, "--token", "seo_test", "--project", "taffy", "--quiet"}, args...))
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	output.Stdout = os.Stdout
	flagDryRun = false

	if len(bodies) != 2 || paths[0] != "POST /api/v1/projects/taffy/reviews" {
		t.Fatalf("calls: %v", paths)
	}
	first := bodies[0]
	if first["plan_item_id"] != float64(7) || first["draft_format"] != "html" || !strings.Contains(first["draft"].(string), "Iron helps") {
		t.Errorf("first body: %+v", first)
	}
	if results, ok := first["adapter_results"].([]any); !ok || len(results) != 1 {
		t.Errorf("adapter results not sent: %+v", first["adapter_results"])
	}
	if bodies[1]["dry_run"] != true || bodies[1]["draft_format"] != "markdown" || bodies[1]["plan_item_id"] != nil {
		t.Errorf("dry run body: %+v", bodies[1])
	}
	if len(exits) != 1 || exits[0] != output.ExitError {
		t.Errorf("want one exit 1 for the fail and none for the estimate, got %v", exits)
	}
}

func TestQaCommandNeedsAFile(t *testing.T) {
	t.Setenv(config.EnvConfigDir, t.TempDir())
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetArgs([]string{"--token", "seo_test", "--project", "taffy", "qa", "--item", "7"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--file PATH") {
		t.Fatalf("want a usage error naming --file, got %v", err)
	}
}
