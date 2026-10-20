package cli

import "github.com/spf13/cobra"

func watchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "watch every live page: index on day 7 and 14, impressions on day 28",
		RunE: func(_ *cobra.Command, _ []string) error {
			return usageError("seo watch is not built yet")
		},
	}
}
