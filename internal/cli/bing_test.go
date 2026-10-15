package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBingAndDemandLines(t *testing.T) {
	var bing map[string]any
	if err := json.Unmarshal([]byte(`{"status":"ok","checked_on":"2026-10-15","summary":{"checked":3,"bing_top10":1,"google_top10":2},
		"disagreements":[{"keyword":"ferritin levels by age","google":3,"bing":18,"gap":15}],
		"google_only":[{"keyword":"iron panel","google":8}],"run":{"checked":3},"cost":0.0105}`), &bing); err != nil {
		t.Fatal(err)
	}
	out := renderBing(bing)
	for _, want := range []string{"Bing checked 2026-10-15", "disagree by 10+", "ferritin levels by age", "not ranked on Bing", "iron panel", "checked 3"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if got := renderBing(map[string]any{"status": "none"}); !strings.Contains(got, "seo track bing --run") {
		t.Errorf("empty state: %s", got)
	}
	var demand map[string]any
	if err := json.Unmarshal([]byte(`{"prompts":[{"demand":89,"text":"which tool searches youtube comments","matched":"youtube comment search","locked":false}],"suggestion":"1 locked prompts show no AI searches."}`), &demand); err != nil {
		t.Fatal(err)
	}
	out = renderDemand(demand)
	for _, want := range []string{"89", "youtube comment search", "no AI searches"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if !IsPaid("track") || !IsPaid("ai") {
		t.Error("track and ai must stay paid so seo estimate covers bing and demand")
	}
}
