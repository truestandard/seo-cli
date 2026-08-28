package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	sinceDays = regexp.MustCompile(`^(\d+)d$`)
	sinceDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func since(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if sinceDays.MatchString(trimmed) {
		return trimmed, nil
	}
	if sinceDate.MatchString(trimmed) {
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return "", usageError("invalid date in --since: " + value)
		}
		return trimmed, nil
	}
	return "", usageError(fmt.Sprintf("invalid --since value %q: use 7d, 30d, or YYYY-MM-DD", value))
}

func exactArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != n {
			return usageError("usage: " + usage)
		}
		return nil
	}
}

func minArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) < n {
			return usageError("usage: " + usage)
		}
		return nil
	}
}

func maxArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > n {
			return usageError("usage: " + usage)
		}
		return nil
	}
}

func readText(args []string, file string) (string, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", usageError(err.Error())
		}
		return string(data), nil
	}
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

func body(fields map[string]any) map[string]any {
	payload := map[string]any{}
	for key, value := range fields {
		if value == nil {
			continue
		}
		payload[key] = value
	}
	if flagDryRun {
		payload["dry_run"] = true
	}
	return payload
}

func optString(cmd *cobra.Command, name string) any {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, _ := cmd.Flags().GetString(name)
	return value
}

func optInt(cmd *cobra.Command, name string) any {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, _ := cmd.Flags().GetInt(name)
	return value
}

func optBool(cmd *cobra.Command, name string) any {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, _ := cmd.Flags().GetBool(name)
	return value
}

func optTrue(cmd *cobra.Command, name string) any {
	value, _ := cmd.Flags().GetBool(name)
	if !value {
		return nil
	}
	return true
}

func pluralize(count int, singular string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %ss", count, singular)
}
