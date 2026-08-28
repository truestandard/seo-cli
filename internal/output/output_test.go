package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/truestandard/seo-cli/internal/api"
)

func TestExitFor(t *testing.T) {
	cases := map[string]struct {
		err  *api.APIError
		want int
	}{
		"unauthorized":      {&api.APIError{Code: "unauthorized", Status: 401}, ExitNotAuthenticated},
		"not_authenticated": {&api.APIError{Code: "not_authenticated"}, ExitNotAuthenticated},
		"usage":             {&api.APIError{Code: "usage"}, ExitUsage},
		"credits":           {&api.APIError{Code: "insufficient_credits", Status: 402}, ExitInsufficientCredits},
		"rate":              {&api.APIError{Code: "rate_limited", Status: 429}, ExitRateLimited},
		"provider":          {&api.APIError{Code: "provider_error", Status: 502}, ExitServerError},
		"not_found":         {&api.APIError{Code: "not_found", Status: 404}, ExitError},
	}
	for name, c := range cases {
		if got := ExitFor(c.err); got != c.want {
			t.Errorf("%s: got %d want %d", name, got, c.want)
		}
	}
}

func TestEnvelopeCarriesExtra(t *testing.T) {
	obj, code := Envelope(&api.APIError{Code: "insufficient_credits", Message: "no", Extra: map[string]any{"checkout_url": "u"}})
	if code != ExitInsufficientCredits || obj["error"] != "insufficient_credits" || obj["checkout_url"] != "u" {
		t.Fatalf("got %v %d", obj, code)
	}
	obj, code = Envelope(errors.New("plain"))
	if code != ExitError || obj["error"] != "error" || obj["message"] != "plain" {
		t.Fatalf("got %v %d", obj, code)
	}
}

func TestEmitDoesNotEscapeHTML(t *testing.T) {
	var buf bytes.Buffer
	Stdout = &buf
	defer func() { Stdout = nil }()
	Emit(map[string]any{"url": "https://a.b/?x=1&y=2"})
	if !bytes.Contains(buf.Bytes(), []byte("&y=2")) {
		t.Fatalf("escaped: %s", buf.String())
	}
}

func TestRenderPrettyUsesHuman(t *testing.T) {
	var buf bytes.Buffer
	Stdout = &buf
	Pretty = true
	defer func() { Stdout = nil; Pretty = false }()
	Render(map[string]any{"a": 1}, func() string { return "human" })
	if buf.String() != "human\n" {
		t.Fatalf("got %q", buf.String())
	}
}
