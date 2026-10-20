package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/table"
)

const defaultProfileCopy = ".claude/seo-profile.md"

func profileCmd() *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "show, edit, import or pull this product's profile (free)"}
	show := profileShowCmd()
	c.RunE = show.RunE
	c.Args = exactArgs(0, "seo profile [show | set --file data.json [--replace] | import FILE.yml | pull [--out PATH]]")
	c.AddCommand(show, profileSetCmd(), profileImportCmd(), profilePullCmd())
	return c
}

func profileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "print the effective profile, its version and digest",
		Args:  exactArgs(0, "seo profile show"),
		RunE: func(_ *cobra.Command, _ []string) error {
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/profile"), nil)
			if err != nil {
				return err
			}
			output.Render(data, func() string { return joinLines(profileLines(table.AsObject(data)), "\n") })
			return nil
		},
	}
}

func profileSetCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "set --file data.json",
		Short: "merge profile keys from a JSON file into the saved profile (--replace swaps it)",
		Args:  exactArgs(0, "seo profile set --file data.json [--replace]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			file, _ := cmd.Flags().GetString("file")
			if file == "" {
				return usageError("usage: seo profile set --file data.json [--replace]")
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				return usageError(err.Error())
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				return usageError(fmt.Sprintf("%s is not a JSON object: %v", file, err))
			}
			replace, _ := cmd.Flags().GetBool("replace")
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			result, err := client.Patch(api.ProjectPath(slug, "/profile"), map[string]any{"data": data, "merge": !replace})
			if err != nil {
				return err
			}
			output.Render(result, func() string { return joinLines(profileLines(table.AsObject(result)), "\n") })
			return nil
		},
	}
	c.Flags().String("file", "", "JSON file with the profile keys to set")
	c.Flags().Bool("replace", false, "replace the saved profile instead of merging into it")
	return c
}

func profileImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import FILE.yml",
		Short: "replace the profile from a YAML file (profile data, or a db/profiles file for this project)",
		Args:  exactArgs(1, "seo profile import FILE.yml"),
		RunE: func(_ *cobra.Command, args []string) error {
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return usageError(err.Error())
			}
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			result, err := client.Post(api.ProjectPath(slug, "/profile/import"), map[string]any{"yaml": string(raw)})
			if err != nil {
				return err
			}
			output.Render(result, func() string { return joinLines(profileLines(table.AsObject(result)), "\n") })
			return nil
		},
	}
}

func profilePullCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "pull",
		Short: "write a read-only markdown copy of the profile into this repo for agents",
		Args:  exactArgs(0, "seo profile pull [--out .claude/seo-profile.md]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, _ := cmd.Flags().GetString("out")
			client, slug, err := projectClient()
			if err != nil {
				return err
			}
			data, err := client.Get(api.ProjectPath(slug, "/profile"), nil)
			if err != nil {
				return err
			}
			root := table.AsObject(data)
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(out, []byte(profileMarkdown(root, time.Now().Format("2006-01-02"))), 0o644); err != nil {
				return err
			}
			result := map[string]any{"path": out, "project": root["project"], "version": root["version"], "digest": root["digest"]}
			output.Render(result, func() string {
				return fmt.Sprintf("wrote %s (profile version %v). Edit the profile in the seo app, not this file.", out, orDash(root["version"]))
			})
			return nil
		},
	}
	c.Flags().String("out", defaultProfileCopy, "where to write the copy")
	return c
}

func profileLines(root table.Row) []string {
	effective := table.AsObject(root["effective"])
	voice := table.AsObject(effective["voice"])
	facts := table.AsObject(effective["facts"])
	qa := table.AsObject(effective["qa"])
	head := fmt.Sprintf("profile version %v for %v (digest %s)", orDash(root["version"]), root["project"], shortDigest(root["digest"]))
	if root["version"] == nil {
		head = fmt.Sprintf("no saved profile for %v yet; the defaults apply (digest %s)", root["project"], shortDigest(root["digest"]))
	}
	if changed, ok := root["changed"].(bool); ok {
		if changed {
			head = fmt.Sprintf("saved profile version %v (digest %s)", root["version"], shortDigest(root["digest"]))
		} else {
			head = fmt.Sprintf("nothing changed; profile stays at version %v", root["version"])
		}
	}
	lines := []string{
		head,
		fmt.Sprintf("reader: %v; grade %v or lower; banned words %d; approved facts %d; banned claims %d", orDash(voice["reader"]), voice["grade_max"], listLen(voice["banned_words"]), listLen(facts["approved"]), listLen(facts["banned_claims"])),
		"page types: " + strings.Join(stringList(effective["page_types"]), ", "),
		fmt.Sprintf("pass at %v of 40, no area under %v, at most %v rounds", qa["pass_score"], qa["min_area"], qa["max_attempts"]),
	}
	if criteria := table.AsRows(qa["criteria"]); len(criteria) > 0 {
		lines = append(lines, table.Table(criteria, []string{"id", "area", "scope", "hard_fail"}, nil))
	}
	if adapters := table.AsRows(effective["adapters"]); len(adapters) > 0 {
		lines = append(lines, "adapters", table.Table(adapters, []string{"name", "stage", "hard_fail", "command"}, nil))
	}
	return lines
}

func shortDigest(value any) string {
	digest, _ := value.(string)
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func profileMarkdown(root table.Row, generatedOn string) string {
	effective := table.AsObject(root["effective"])
	voice := table.AsObject(effective["voice"])
	facts := table.AsObject(effective["facts"])
	cta := table.AsObject(effective["cta"])
	qa := table.AsObject(effective["qa"])
	links := table.AsObject(effective["links"])
	repo := table.AsObject(effective["repo"])
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	line("<!-- Generated by `seo profile pull`. Read-only copy. Do not edit. -->")
	line("# SEO profile: %v", root["project"])
	line("")
	line("Generated on %s from the seo app: profile version %v, digest %s.", generatedOn, orDash(root["version"]), shortDigest(root["digest"]))
	line("This file is a read-only copy for agents. Edits here change nothing. Edit the profile in the seo app with `seo profile set --file data.json` or `seo profile import FILE.yml`, then run `seo profile pull` again.")
	line("")
	line("## Voice")
	line("")
	line("- Reader: %v", orDash(voice["reader"]))
	line("- Tone: %v", orDash(voice["tone"]))
	line("- Grade: %v or lower", orDash(voice["grade_max"]))
	line("- Person: %v", orDash(voice["person"]))
	line("- Banned words: %s", listOrNone(voice["banned_words"]))
	line("- Banned punctuation: %s", listOrNone(voice["banned_punctuation"]))
	line("")
	line("## Facts")
	line("")
	approved := table.AsRows(facts["approved"])
	if len(approved) == 0 {
		line("Approved facts: none yet. Make no product claim without a source.")
	} else {
		line("Approved facts (the only product claims you may make without another source):")
		line("")
		for _, fact := range approved {
			if source, ok := fact["source"].(string); ok && source != "" {
				line("- %v (source: %s)", fact["claim"], source)
			} else {
				line("- %v", fact["claim"])
			}
		}
	}
	line("")
	line("- Banned claims: %s", listOrNone(facts["banned_claims"]))
	line("- Pricing: %v", orDash(facts["pricing"]))
	line("- Free or trial: %v", orDash(facts["free_or_trial"]))
	line("")
	line("## Calls to action")
	line("")
	line("- Default: %s", ctaText(cta["default"]))
	byFormat := table.AsObject(cta["by_format"])
	for _, format := range sortedRowKeys(byFormat) {
		line("- %s: %s", format, ctaText(byFormat[format]))
	}
	line("")
	line("## Page types")
	line("")
	line("%s", listOrNone(effective["page_types"]))
	line("")
	line("## Checks")
	line("")
	line("Pass at %v of 40 with no area under %v. At most %v rounds, then a person. Disabled: %s.", qa["pass_score"], qa["min_area"], qa["max_attempts"], listOrNone(qa["disabled"]))
	line("")
	line("| id | area | scope | hard fail | question |")
	line("|---|---|---|---|---|")
	for _, criterion := range table.AsRows(qa["criteria"]) {
		line("| %v | %v | %v | %v | %v |", criterion["id"], criterion["area"], criterion["scope"], criterion["hard_fail"], markdownCell(criterion["question"]))
	}
	line("")
	line("## Adapters")
	line("")
	adapters := table.AsRows(effective["adapters"])
	if len(adapters) == 0 {
		line("None.")
	} else {
		line("| name | stage | hard fail | command |")
		line("|---|---|---|---|")
		for _, adapter := range adapters {
			line("| %v | %v | %v | `%v` |", markdownCell(adapter["name"]), adapter["stage"], adapter["hard_fail"], markdownCell(adapter["command"]))
		}
	}
	line("")
	line("## Links")
	line("")
	line("- Inbound links needed: %v", orDash(links["min_inbound"]))
	line("- Hubs: %s", listOrNone(links["hubs"]))
	line("")
	line("## Repo")
	line("")
	line("- Path: %v", orDash(repo["path"]))
	line("- Pages: %v", orDash(repo["pages_glob"]))
	line("- Sitemap: %v", orDash(repo["sitemap_url"]))
	line("")
	line("## Full profile")
	line("")
	full, _ := json.MarshalIndent(effective, "", "  ")
	line("```json")
	line("%s", full)
	line("```")
	return b.String()
}

func listOrNone(value any) string {
	items := stringList(value)
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func ctaText(value any) string {
	target := table.AsObject(value)
	if target == nil {
		return "none"
	}
	return fmt.Sprintf("%v (%v)", target["label"], target["path"])
}

func markdownCell(value any) string {
	return strings.ReplaceAll(fmt.Sprint(orDash(value)), "|", "\\|")
}

func sortedRowKeys(row table.Row) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
