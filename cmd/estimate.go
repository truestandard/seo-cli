package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var PaidCommands = []string{"research", "serp", "track", "ai", "floor", "backlinks", "domain", "mentions", "audit"}

func IsPaid(name string) bool {
	for _, paid := range PaidCommands {
		if paid == name {
			return true
		}
	}
	return false
}

func estimateCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "estimate <command> [args...]",
		Short:              "re-run a paid command with --dry-run and print the estimate (" + strings.Join(PaidCommands, ", ") + ")",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || strings.HasPrefix(args[0], "-") {
				return usageError("usage: seo estimate <" + strings.Join(PaidCommands, "|") + "> [args...]")
			}
			if !IsPaid(args[0]) {
				return usageError("estimate only applies to paid commands: " + strings.Join(PaidCommands, ", "))
			}
			forwarded := forwardedGlobals()
			nested := newRootCmd()
			nested.SetArgs(append(append(args, forwarded...), "--dry-run"))
			return nested.Execute()
		},
	}
}

func forwardedGlobals() []string {
	var args []string
	if flagProject != "" {
		args = append(args, "--project", flagProject)
	}
	if flagAPIURL != "" {
		args = append(args, "--api-url", flagAPIURL)
	}
	if flagToken != "" {
		args = append(args, "--token", flagToken)
	}
	if flagJSON {
		args = append(args, "--json")
	}
	if flagPretty {
		args = append(args, "--pretty")
	}
	if flagQuiet {
		args = append(args, "--quiet")
	}
	return args
}
