package cli

import "github.com/spf13/cobra"

func briefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brief",
		Short: "build or show the brief for one ticket",
		RunE: func(_ *cobra.Command, _ []string) error {
			return usageError("seo brief is not built yet")
		},
	}
}
