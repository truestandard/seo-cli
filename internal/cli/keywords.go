package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

var keywordHeaders = map[string]string{"set_name": "set", "target_path": "path"}

func keywordsCmd() *cobra.Command {
	c := &cobra.Command{Use: "keywords", Short: "list, add, update or remove tracked keywords"}
	list := keywordsListCmd()
	c.RunE = list.RunE
	c.Args = exactArgs(0, "seo keywords [list | add | update | remove]")
	c.Flags().AddFlagSet(list.Flags())
	c.AddCommand(list, keywordsAddCmd(), keywordsUpdateCmd(), keywordsRemoveCmd())
	return c
}

func keywordsListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list keywords",
		Args:  exactArgs(0, "seo keywords list [--set <name>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			set, _ := cmd.Flags().GetString("set")
			data, err := client.Get(api.ProjectPath(slug, "/keywords"), api.Query{"set": set})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "keywords")
				if len(rows) == 0 {
					return "no keywords"
				}
				return table.Table(rows, []string{"id", "keyword", "set_name", "track", "target_path", "volume", "kd", "locked"}, keywordHeaders)
			})
			return nil
		},
	}
	c.Flags().String("set", "", "filter by set name")
	return c
}

func keywordsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <keywords...>",
		Short: "add keywords",
		Args:  minArgs(1, "seo keywords add <keywords...> [--track --path --set --locked --volume --kd]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			var keywords []any
			for _, text := range args {
				keywords = append(keywords, map[string]any{
					"keyword":     text,
					"track":       optString(cmd, "track"),
					"target_path": optString(cmd, "path"),
					"set_name":    optString(cmd, "set"),
					"locked":      optBool(cmd, "locked"),
					"volume":      optInt(cmd, "volume"),
					"kd":          optInt(cmd, "kd"),
				})
			}
			data, err := client.Post(api.ProjectPath(slug, "/keywords"), map[string]any{"keywords": compactAll(keywords)})
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				rows := table.AsRows(data, "keywords")
				if len(rows) == 0 {
					return "added " + pluralize(len(args), "keyword")
				}
				return table.Table(rows, []string{"id", "keyword", "set_name", "track", "target_path", "locked"}, keywordHeaders)
			})
			return nil
		},
	}
	c.Flags().String("track", "", "track label, e.g. bofu, brand")
	c.Flags().String("path", "", "target path on the site")
	c.Flags().String("set", "", "set name (default guarantee)")
	c.Flags().Bool("locked", false, "lock the keywords as a measurement contract")
	c.Flags().Int("volume", 0, "monthly volume if already known")
	c.Flags().Int("kd", 0, "keyword difficulty if already known")
	return c
}

func keywordsUpdateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "update a keyword's target path, track, or status (locked keywords accept status only)",
		Args:  exactArgs(1, "seo keywords update <id> [--path --track --status]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			patch := compact(map[string]any{
				"target_path": optString(cmd, "path"),
				"track":       optString(cmd, "track"),
				"status":      optString(cmd, "status"),
			})
			if len(patch) == 0 {
				return usageError("pass at least one of --path, --track, --status")
			}
			data, err := client.Patch(api.ProjectPath(slug, "/keywords/"+args[0]), patch)
			if err != nil {
				return err
			}
			output.Render(data, func() string {
				row := table.First(table.AsObject(data), "keyword")
				if len(row) == 0 {
					return "updated " + args[0]
				}
				return table.Table([]table.Row{row}, []string{"id", "keyword", "set_name", "status", "track", "target_path", "locked"}, keywordHeaders)
			})
			return nil
		},
	}
	c.Flags().String("path", "", "target path on the site")
	c.Flags().String("track", "", "track label, e.g. bofu, brand")
	c.Flags().String("status", "", "tracked, paused, or retired")
	return c
}

func keywordsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <ids...>",
		Short: "remove keywords by id",
		Args:  minArgs(1, "seo keywords remove <ids...>"),
		RunE: func(_ *cobra.Command, args []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			responses := []any{}
			for _, id := range args {
				response, err := client.Delete(api.ProjectPath(slug, "/keywords/"+id))
				if err != nil {
					return err
				}
				responses = append(responses, response)
			}
			output.Render(map[string]any{"removed": args, "responses": responses}, func() string {
				return "removed " + strings.Join(args, ", ")
			})
			return nil
		},
	}
}

func compact(fields map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range fields {
		if value != nil {
			out[key] = value
		}
	}
	return out
}

func compactAll(items []any) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = compact(item.(map[string]any))
	}
	return out
}
