package api

import "testing"

func TestParseBackendShape(t *testing.T) {
	e := ParseAPIError(401, []byte(`{"error":"unauthorized","code":"unauthorized"}`))
	if e.Code != "unauthorized" || e.Message != "unauthorized" || e.Status != 401 {
		t.Fatalf("got %+v", e)
	}
	e = ParseAPIError(404, []byte(`{"error":"Couldn't find Project","code":"not_found"}`))
	if e.Code != "not_found" || e.Message != "Couldn't find Project" {
		t.Fatalf("got %+v", e)
	}
}

func TestParseStandardShapeKeepsExtra(t *testing.T) {
	e := ParseAPIError(402, []byte(`{"error":"insufficient_credits","message":"Not enough.","balance":0,"checkout_url":"https://x"}`))
	if e.Code != "insufficient_credits" || e.Message != "Not enough." {
		t.Fatalf("got %+v", e)
	}
	if e.Extra["checkout_url"] != "https://x" || e.Extra["balance"] != float64(0) {
		t.Fatalf("extra lost: %+v", e.Extra)
	}
}

func TestParseHTMLBody(t *testing.T) {
	e := ParseAPIError(502, []byte("<html>bad gateway</html>"))
	if e.Code != "http_error" || e.Message != "HTTP 502" || e.Status != 502 {
		t.Fatalf("got %+v", e)
	}
}

func TestLastSseData(t *testing.T) {
	got := string(lastSseData([]byte("event: message\ndata: {\"a\":1}\n\nevent: message\ndata: {\"b\":2}\n\n")))
	if got != `{"b":2}` {
		t.Fatalf("got %q", got)
	}
}
