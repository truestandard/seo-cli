package cli

import "github.com/spf13/cobra"

func qaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "qa",
		Short: "check a draft against the brief and the product profile",
		RunE: func(_ *cobra.Command, _ []string) error {
			return usageError("seo qa is not built yet")
		},
	}
}
