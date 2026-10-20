package cli

import "github.com/spf13/cobra"

func profileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "show, edit or import this product's profile",
		RunE: func(_ *cobra.Command, _ []string) error {
			return usageError("seo profile is not built yet")
		},
	}
}
