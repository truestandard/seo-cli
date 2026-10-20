package cli

import "github.com/spf13/cobra"

func itemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "items",
		Short: "tickets: every planned page and its status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return usageError("seo items is not built yet")
		},
	}
}
