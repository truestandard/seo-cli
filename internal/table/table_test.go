package table

import "testing"

func TestTableAlignsNumbersRight(t *testing.T) {
	rows := []Row{{"keyword": "seo cli", "pos": float64(3)}, {"keyword": "rank tracker api", "pos": float64(12)}}
	got := Table(rows, []string{"keyword", "pos"}, nil)
	want := "keyword           pos\nseo cli             3\nrank tracker api   12"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestCell(t *testing.T) {
	cases := map[any]string{nil: "-", true: "yes", false: "no", float64(2): "2", float64(2.5): "2.5", "": "-", "x": "x"}
	for in, want := range cases {
		if got := Cell(in); got != want {
			t.Errorf("%v: got %q want %q", in, got, want)
		}
	}
	if got := Cell([]any{"a", float64(1)}); got != "a,1" {
		t.Errorf("list: %q", got)
	}
}

func TestMoney(t *testing.T) {
	if got := Money(0.006); got != "$0.006" {
		t.Errorf("got %q", got)
	}
	if got := Money(float64(2)); got != "$2.0" {
		t.Errorf("got %q", got)
	}
	if got := Money("0.25"); got != "$0.25" {
		t.Errorf("got %q", got)
	}
}

func TestEstimateLine(t *testing.T) {
	data := map[string]any{"estimate": map[string]any{"cost": 0.06, "keywords": float64(100)}}
	if !IsEstimate(data) {
		t.Fatal("expected an estimate")
	}
	if got := EstimateLine(data); got != "estimate $0.06 (keywords=100), nothing spent" {
		t.Fatalf("got %q", got)
	}
}

func TestAsRowsFindsNestedList(t *testing.T) {
	data := map[string]any{"keywords": []any{map[string]any{"id": float64(1)}, "junk"}}
	if rows := AsRows(data, "keywords"); len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows := AsRows([]any{map[string]any{"a": 1}}); len(rows) != 1 {
		t.Fatalf("got %d rows", len(rows))
	}
	if rows := AsRows("nope"); rows != nil {
		t.Fatal("expected nil")
	}
}

func TestKeyValueBlockSkipsNil(t *testing.T) {
	got := KeyValueBlock([]Pair{{"id", float64(4)}, {"gone", nil}, {"status", "done"}})
	if got != "id      4\nstatus  done" {
		t.Fatalf("got %q", got)
	}
}
