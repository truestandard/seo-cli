package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/skills"
)

const noAgents = "no coding agents detected here (looked for .claude/, AGENTS.md, .codex/)"

func skillsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "skills",
		Short: "install the seo skill into coding agents in this project (Claude Code, Codex)",
		Long: "Writes a managed skill that teaches an agent the command list, the stdout contract, " +
			"the exit codes and the dry-run rule. Touches only .claude/skills/seo/SKILL.md and the fenced block in AGENTS.md.",
	}
	c.AddCommand(skillsInstallCmd(), skillsUninstallCmd(), skillsListCmd(), skillsDoctorCmd())
	return c
}

func skillsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "install or refresh the skill for the detected agents",
		Args:  exactArgs(0, "seo skills install"),
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			results, err := skills.Install(dir)
			if err != nil {
				return err
			}
			output.Render(map[string]any{"results": results}, func() string { return renderResults(results) })
			return nil
		},
	}
}

func skillsUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "remove the installed skill from the detected agents",
		Args:  exactArgs(0, "seo skills uninstall"),
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			results, err := skills.Uninstall(dir)
			if err != nil {
				return err
			}
			output.Render(map[string]any{"results": results}, func() string { return renderResults(results) })
			return nil
		},
	}
}

func skillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "show install status across the detected agents",
		Args:  exactArgs(0, "seo skills list"),
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			statuses := skills.List(dir)
			output.Render(map[string]any{"targets": statuses}, func() string { return renderStatuses(statuses, "seo skills doctor --fix") })
			return nil
		},
	}
}

func skillsDoctorCmd() *cobra.Command {
	var fix bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "find stale or missing skills; --fix refreshes them in place",
		Args:  exactArgs(0, "seo skills doctor [--fix]"),
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			report, err := skills.Doctor(dir, fix)
			if err != nil {
				return err
			}
			output.Render(report, func() string {
				text := renderStatuses(report.Targets, "seo skills doctor --fix")
				if fix && len(report.Fixed) > 0 {
					text += fmt.Sprintf("\nrefreshed %d target(s)", len(report.Fixed))
				}
				return text
			})
			return nil
		},
	}
	c.Flags().BoolVar(&fix, "fix", false, "refresh stale or missing skills in place")
	return c
}

func renderResults(results []skills.Result) string {
	if len(results) == 0 {
		return noAgents
	}
	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("%-12s %s: %s", r.Agent, r.Path, r.Action))
	}
	return strings.Join(lines, "\n")
}

func renderStatuses(statuses []skills.Status, fixCommand string) string {
	if len(statuses) == 0 {
		return noAgents
	}
	var lines []string
	for _, s := range statuses {
		state := "not installed"
		switch {
		case s.Installed && s.Stale:
			state = "stale (run: " + fixCommand + ")"
		case s.Installed:
			state = "installed"
		}
		lines = append(lines, fmt.Sprintf("%-12s %s: %s", s.Agent, s.Path, state))
	}
	return strings.Join(lines, "\n")
}
