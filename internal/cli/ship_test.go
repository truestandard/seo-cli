package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderShipShowsItemAndWatch(t *testing.T) {
	var data any
	raw := `{"ship_event":{"id":7,"url":"https://precis.health/guides/iron","verified":true,"http_status":200,"h1":"Iron","plan_item_id":3},
		"plan_item":{"id":3,"kind":"new_page","status":"watching"},"page_watch_id":11}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderShip(data, "https://precis.health/guides/iron", nil)
	for _, want := range []string{"verified=yes", "item=3", "status=watching", "watch=11"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "not shipped") {
		t.Errorf("verified ship printed a failure:\n%s", out)
	}
}

func TestRenderShipSaysWhyItFailed(t *testing.T) {
	var data any
	raw := `{"ship_event":{"id":8,"url":"https://precis.health/guides/zinc","verified":false,"http_status":404},
		"plan_item":{"id":4,"kind":"new_page","status":"approved"},"reason":"The page answered 404 instead of 200."}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	out := renderShip(data, "https://precis.health/guides/zinc", "zinc")
	for _, want := range []string{"verified=no", "status=approved", "not shipped: The page answered 404", "keyword=zinc"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
