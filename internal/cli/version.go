package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/output"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the CLI version",
		Args:  exactArgs(0, "seo version"),
		RunE: func(_ *cobra.Command, _ []string) error {
			output.Render(map[string]any{
				"version": version, "commit": commit, "date": date,
				"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH,
			}, func() string {
				return fmt.Sprintf("seo %s (commit %s, built %s) %s/%s", version, commit, date, runtime.GOOS, runtime.GOARCH)
			})
			return nil
		},
	}
}
