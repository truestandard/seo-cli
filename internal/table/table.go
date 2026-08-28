package table

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Row = map[string]any

type Pair struct {
	Key   string
	Value any
}

var numericText = regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`)

func Cell(value any) string {
	switch v := value.(type) {
	case nil:
		return "-"
	case bool:
		if v {
			return "yes"
		}
		return "no"
	case float64:
		return formatNumber(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		if v == "" {
			return "-"
		}
		return v
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = Cell(item)
		}
		return strings.Join(parts, ",")
	case json.Number:
		return v.String()
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func formatNumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	s := strconv.FormatFloat(v, 'f', 4, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func isNumeric(value any) bool {
	switch v := value.(type) {
	case float64, int, int64, json.Number:
		return true
	case string:
		return numericText.MatchString(v)
	}
	return false
}

func Table(rows []Row, columns []string, headers map[string]string) string {
	if len(rows) == 0 {
		return ""
	}
	keys := columns
	if len(keys) == 0 {
		keys = uniqueKeys(rows)
	}
	cells := make([][]string, len(rows))
	for i, row := range rows {
		cells[i] = make([]string, len(keys))
		for j, key := range keys {
			cells[i][j] = Cell(row[key])
		}
	}
	titles := make([]string, len(keys))
	widths := make([]int, len(keys))
	rightAligned := make([]bool, len(keys))
	for j, key := range keys {
		titles[j] = key
		if h, ok := headers[key]; ok {
			titles[j] = h
		}
		widths[j] = len([]rune(titles[j]))
		for i := range rows {
			if w := len([]rune(cells[i][j])); w > widths[j] {
				widths[j] = w
			}
			if isNumeric(rows[i][key]) {
				rightAligned[j] = true
			}
		}
	}
	renderLine := func(line []string) string {
		parts := make([]string, len(line))
		for j, cell := range line {
			if rightAligned[j] {
				parts[j] = padLeft(cell, widths[j])
			} else {
				parts[j] = padRight(cell, widths[j])
			}
		}
		return strings.TrimRight(strings.Join(parts, "  "), " ")
	}
	lines := []string{renderLine(titles)}
	for _, line := range cells {
		lines = append(lines, renderLine(line))
	}
	return strings.Join(lines, "\n")
}

func padLeft(s string, width int) string {
	return strings.Repeat(" ", width-len([]rune(s))) + s
}

func padRight(s string, width int) string {
	return s + strings.Repeat(" ", width-len([]rune(s)))
}

func uniqueKeys(rows []Row) []string {
	var keys []string
	seen := map[string]bool{}
	for _, row := range rows {
		for _, key := range sortedKeys(row) {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func sortedKeys(row Row) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func SummaryLine(pairs []Pair) string {
	var parts []string
	for _, p := range pairs {
		if p.Value == nil {
			continue
		}
		parts = append(parts, p.Key+"="+Cell(p.Value))
	}
	return strings.Join(parts, " ")
}

func SummaryOf(object Row) string {
	var pairs []Pair
	for _, key := range sortedKeys(object) {
		pairs = append(pairs, Pair{key, object[key]})
	}
	return SummaryLine(pairs)
}

func KeyValueBlock(pairs []Pair) string {
	width := 0
	var kept []Pair
	for _, p := range pairs {
		if p.Value == nil {
			continue
		}
		kept = append(kept, p)
		if len(p.Key) > width {
			width = len(p.Key)
		}
	}
	lines := make([]string, len(kept))
	for i, p := range kept {
		lines[i] = padRight(p.Key, width) + "  " + Cell(p.Value)
	}
	return strings.Join(lines, "\n")
}

func Money(value any) string {
	var amount float64
	switch v := value.(type) {
	case float64:
		amount = v
	case int:
		amount = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Cell(value)
		}
		amount = parsed
	default:
		return Cell(value)
	}
	s := strconv.FormatFloat(amount, 'f', 4, 64)
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "0"
	}
	return "$" + s
}

func AsObject(value any) Row {
	object, _ := value.(map[string]any)
	return object
}

func Pick(value any, key string) Row {
	return AsObject(AsObject(value)[key])
}

func AsRows(value any, candidateKeys ...string) []Row {
	if list, ok := value.([]any); ok {
		return objectsOf(list)
	}
	object := AsObject(value)
	if object == nil {
		return nil
	}
	for _, key := range append(candidateKeys, "rows", "results", "data", "items") {
		if list, ok := object[key].([]any); ok {
			return objectsOf(list)
		}
	}
	return nil
}

func objectsOf(list []any) []Row {
	rows := make([]Row, 0, len(list))
	for _, item := range list {
		if object := AsObject(item); object != nil {
			rows = append(rows, object)
		}
	}
	return rows
}

func IsEstimate(data any) bool {
	return Pick(data, "estimate") != nil
}

func EstimateLine(data any) string {
	estimate := Pick(data, "estimate")
	if estimate == nil {
		estimate = AsObject(data)
	}
	var rest []Pair
	for _, key := range sortedKeys(estimate) {
		if key != "cost" {
			rest = append(rest, Pair{key, estimate[key]})
		}
	}
	line := "estimate " + Money(estimate["cost"])
	if detail := SummaryLine(rest); detail != "" {
		line += " (" + detail + ")"
	}
	return line + ", nothing spent"
}

func First(object Row, keys ...string) Row {
	for _, key := range keys {
		if nested := AsObject(object[key]); nested != nil {
			return nested
		}
	}
	return object
}
