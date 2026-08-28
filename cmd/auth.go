package cmd

import (
	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/auth"
	"github.com/truestandard/seo-cli/internal/config"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func authCmd() *cobra.Command {
	c := &cobra.Command{Use: "auth", Short: "log in with an API key, check the login, or log out"}
	c.AddCommand(authLoginCmd(), authLogoutCmd(), authStatusCmd())
	return c
}

func authLoginCmd() *cobra.Command {
	var noSkills bool
	c := &cobra.Command{
		Use:   "login --token seo_…",
		Short: "check the key against whoami and save it (0600)",
		Args:  exactArgs(0, "seo auth login --token seo_… [--api-url <url>] [--no-skills]"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return auth.LoginWithToken(cfg, noSkills)
		},
	}
	c.Flags().BoolVar(&noSkills, "no-skills", false, "skip installing the agent skill into this project")
	return c
}

func loginCmd() *cobra.Command {
	c := authLoginCmd()
	c.Use = "login --token seo_…"
	c.Hidden = true
	return c
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "remove the saved key",
		Args:  exactArgs(0, "seo auth logout"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return auth.Logout()
		},
	}
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "show where the key points and whether the backend accepts it",
		Args:  exactArgs(0, "seo auth status"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if cfg.Token == "" {
				output.Render(map[string]any{"authenticated": false, "api_url": cfg.APIURL}, func() string {
					return "not logged in: run `seo auth login --token seo_…`"
				})
				return nil
			}
			client, err := requireClient()
			if err != nil {
				return err
			}
			identity, err := client.Whoami()
			if err != nil {
				return err
			}
			output.Render(map[string]any{
				"authenticated": true,
				"api_url":       cfg.APIURL,
				"stored":        config.HasStoredToken(),
				"project":       cfg.Project,
				"whoami":        identity,
			}, func() string {
				return "logged in to " + cfg.APIURL + " " + table.SummaryOf(identity)
			})
			return nil
		},
	}
}

func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "the key's identity as the backend sees it",
		Args:  exactArgs(0, "seo whoami"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			identity, err := client.Whoami()
			if err != nil {
				return err
			}
			output.Render(identity, func() string { return table.KeyValueBlock(pairsOf(identity)) })
			return nil
		},
	}
}

func pairsOf(object table.Row) []table.Pair {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	pairs := make([]table.Pair, len(keys))
	for i, key := range keys {
		pairs[i] = table.Pair{Key: key, Value: object[key]}
	}
	return pairs
}
