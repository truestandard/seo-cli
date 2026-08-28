package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/truestandard/seo-cli/internal/api"
)

var (
	Pretty bool
	Quiet  bool
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

const (
	ExitOK                  = 0
	ExitError               = 1
	ExitUsage               = 2
	ExitNeedsClarification  = 10
	ExitNotAuthenticated    = 11
	ExitInsufficientCredits = 12
	ExitRateLimited         = 13
	ExitServerError         = 14
)

func Emit(v any) {
	enc := json.NewEncoder(Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func Text(s string) {
	if s == "" {
		return
	}
	if s[len(s)-1] == '\n' {
		fmt.Fprint(Stdout, s)
		return
	}
	fmt.Fprintln(Stdout, s)
}

func Render(data any, human func() string) {
	if Pretty {
		Text(human())
		return
	}
	Emit(data)
}

func Notice(format string, args ...any) {
	if Quiet {
		return
	}
	fmt.Fprintf(Stderr, format+"\n", args...)
}

func Envelope(err error) (map[string]any, int) {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		obj := map[string]any{"error": apiErr.Code, "message": apiErr.Message}
		for k, v := range apiErr.Extra {
			obj[k] = v
		}
		return obj, ExitFor(apiErr)
	}
	return map[string]any{"error": "error", "message": err.Error()}, ExitError
}

func DieErr(err error) {
	obj, code := Envelope(err)
	Emit(obj)
	os.Exit(code)
}

func ExitFor(e *api.APIError) int {
	switch e.Code {
	case "unauthorized", "not_authenticated", "forbidden":
		return ExitNotAuthenticated
	case "needs_clarification":
		return ExitNeedsClarification
	case "insufficient_credits":
		return ExitInsufficientCredits
	case "rate_limited":
		return ExitRateLimited
	case "usage":
		return ExitUsage
	}
	if e.Status >= 500 {
		return ExitServerError
	}
	return ExitError
}
