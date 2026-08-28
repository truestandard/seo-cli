package cmd

import (
	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/config"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

func projectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "projects",
		Short: "list projects",
		Args:  exactArgs(0, "seo projects"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			data, err := client.Get("/api/v1/projects", nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "projects")
				if len(rows) == 0 {
					return "no projects"
				}
				return table.Table(rows, []string{"slug", "name", "domain", "location_code", "language_code"}, nil)
			})
			return nil
		},
	}
}

func projectCmd() *cobra.Command {
	c := &cobra.Command{Use: "project", Short: "show or create a project"}
	c.AddCommand(projectShowCmd(), projectCreateCmd())
	c.RunE = projectShowCmd().RunE
	c.Args = maxArgs(1, "seo project [show [slug] | create <slug>]")
	return c
}

func projectShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [slug]",
		Short: "show a project (defaults to the current one)",
		Args:  maxArgs(1, "seo project show [slug]"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			slug := ""
			if len(args) == 1 {
				slug = args[0]
			} else if slug, err = requireProject(); err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, ""), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return table.KeyValueBlock(pairsOf(projectFields(data))) })
			return nil
		},
	}
}

func projectCreateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create <slug> --name <name> --domain <domain>",
		Short: "create a project",
		Args:  exactArgs(1, "seo project create <slug> --name <name> --domain <domain>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := requireClient()
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			domain, _ := cmd.Flags().GetString("domain")
			if name == "" || domain == "" {
				return usageError("--name and --domain are required")
			}
			data, err := client.Post("/api/v1/projects", map[string]any{
				"slug":          args[0],
				"name":          name,
				"domain":        domain,
				"repo_path":     optString(cmd, "repo-path"),
				"location_code": optInt(cmd, "location-code"),
				"language_code": optString(cmd, "language-code"),
			})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				return "created project " + args[0] + "\n" + table.KeyValueBlock(pairsOf(projectFields(data)))
			})
			return nil
		},
	}
	c.Flags().String("name", "", "display name")
	c.Flags().String("domain", "", "site domain, e.g. example.com")
	c.Flags().String("repo-path", "", "local repo path")
	c.Flags().Int("location-code", 0, "search location code (default 2840, United States)")
	c.Flags().String("language-code", "", "language code (default en)")
	return c
}

func projectFields(data any) table.Row {
	return table.First(table.AsObject(data), "project")
}

func useCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <slug>",
		Short: "set the default project",
		Args:  exactArgs(1, "seo use <slug>"),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := config.SaveProject(args[0]); err != nil {
				return err
			}
			output.Render(map[string]any{"project": args[0]}, func() string { return "project set to " + args[0] })
			return nil
		},
	}
}
