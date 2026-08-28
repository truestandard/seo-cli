package skills

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/truestandard/seo-cli/skill"
)

const (
	AgentClaude = "claude-code"
	AgentCodex  = "codex"
)

const (
	ActionInstalled    = "installed"
	ActionUpdated      = "updated"
	ActionUnchanged    = "unchanged"
	ActionRemoved      = "removed"
	ActionNotInstalled = "not_installed"
)

const (
	claudeRel = ".claude/skills/seo/SKILL.md"
	agentsRel = "AGENTS.md"
)

type Result struct {
	Agent  string `json:"agent"`
	Path   string `json:"path"`
	Action string `json:"action"`
}

type Status struct {
	Agent     string `json:"agent"`
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	Stale     bool   `json:"stale"`
}

type DoctorReport struct {
	Targets []Status `json:"targets"`
	Fixed   []Result `json:"fixed,omitempty"`
}

func Detect(dir string) []string {
	var agents []string
	if hasClaude(dir) {
		agents = append(agents, AgentClaude)
	}
	if hasCodex(dir) {
		agents = append(agents, AgentCodex)
	}
	return agents
}

func Install(dir string) ([]Result, error) {
	results := []Result{}
	if hasClaude(dir) {
		r, err := installClaude(dir)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	if hasCodex(dir) {
		r, err := installAgents(dir)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	return results, nil
}

func Uninstall(dir string) ([]Result, error) {
	results := []Result{}
	if hasClaude(dir) {
		path := claudeAbs(dir)
		if fileExists(path) {
			if err := os.RemoveAll(filepath.Dir(path)); err != nil {
				return results, err
			}
			results = append(results, Result{Agent: AgentClaude, Path: claudeRel, Action: ActionRemoved})
		} else {
			results = append(results, Result{Agent: AgentClaude, Path: claudeRel, Action: ActionNotInstalled})
		}
	}
	if hasCodex(dir) {
		r, err := uninstallAgents(dir)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	return results, nil
}

func List(dir string) []Status {
	statuses := []Status{}
	if hasClaude(dir) {
		statuses = append(statuses, claudeStatus(dir))
	}
	if hasCodex(dir) {
		statuses = append(statuses, agentsStatus(dir))
	}
	return statuses
}

func Doctor(dir string, fix bool) (DoctorReport, error) {
	report := DoctorReport{Targets: List(dir)}
	if !fix {
		return report, nil
	}
	results, err := Install(dir)
	if err != nil {
		return report, err
	}
	for _, r := range results {
		if r.Action == ActionInstalled || r.Action == ActionUpdated {
			report.Fixed = append(report.Fixed, r)
		}
	}
	report.Targets = List(dir)
	return report, nil
}

func installClaude(dir string) (Result, error) {
	path := claudeAbs(dir)
	want := skill.ClaudeSkill()
	res := Result{Agent: AgentClaude, Path: claudeRel}
	current, err := os.ReadFile(path)
	switch {
	case err == nil && string(current) == want:
		res.Action = ActionUnchanged
		return res, nil
	case err == nil:
		res.Action = ActionUpdated
	default:
		res.Action = ActionInstalled
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return res, err
	}
	return res, os.WriteFile(path, []byte(want), 0o644)
}

func claudeStatus(dir string) Status {
	s := Status{Agent: AgentClaude, Path: claudeRel}
	current, err := os.ReadFile(claudeAbs(dir))
	if err != nil {
		return s
	}
	s.Installed = true
	s.Stale = string(current) != skill.ClaudeSkill()
	return s
}

func installAgents(dir string) (Result, error) {
	path := agentsAbs(dir)
	res := Result{Agent: AgentCodex, Path: agentsRel}
	current := readFileOrEmpty(path)
	present := strings.Contains(current, skill.BeginPrefix)
	next, changed := upsertBlock(current, skill.AgentsBlock())
	switch {
	case !present:
		res.Action = ActionInstalled
	case !changed:
		res.Action = ActionUnchanged
		return res, nil
	default:
		res.Action = ActionUpdated
	}
	return res, os.WriteFile(path, []byte(next), 0o644)
}

func uninstallAgents(dir string) (Result, error) {
	path := agentsAbs(dir)
	res := Result{Agent: AgentCodex, Path: agentsRel, Action: ActionNotInstalled}
	current, err := os.ReadFile(path)
	if err != nil {
		return res, nil
	}
	next, removed := removeBlock(string(current))
	if !removed {
		return res, nil
	}
	res.Action = ActionRemoved
	if strings.TrimSpace(next) == "" {
		return res, os.Remove(path)
	}
	return res, os.WriteFile(path, []byte(next), 0o644)
}

func agentsStatus(dir string) Status {
	s := Status{Agent: AgentCodex, Path: agentsRel}
	current, err := os.ReadFile(agentsAbs(dir))
	if err != nil {
		return s
	}
	block, ok := extractBlock(string(current))
	if !ok {
		return s
	}
	s.Installed = true
	s.Stale = strings.TrimSpace(block) != strings.TrimSpace(skill.AgentsBlock())
	return s
}

func blockBounds(content string) (start, end int, ok bool) {
	bi := strings.Index(content, skill.BeginPrefix)
	if bi < 0 {
		return 0, 0, false
	}
	ei := strings.Index(content, skill.EndMarker)
	if ei < 0 {
		return bi, len(content), true
	}
	return bi, ei + len(skill.EndMarker), true
}

func extractBlock(content string) (string, bool) {
	bi, ei, ok := blockBounds(content)
	if !ok {
		return "", false
	}
	return content[bi:ei], true
}

func upsertBlock(content, block string) (string, bool) {
	bi, ei, ok := blockBounds(content)
	if !ok {
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			return block, true
		}
		return trimmed + "\n\n" + block, true
	}
	before := strings.TrimRight(content[:bi], "\n")
	after := strings.TrimLeft(content[ei:], "\n")
	var sb strings.Builder
	if before != "" {
		sb.WriteString(before)
		sb.WriteString("\n\n")
	}
	sb.WriteString(block)
	if after != "" {
		sb.WriteString("\n")
		sb.WriteString(after)
	}
	next := sb.String()
	return next, next != content
}

func removeBlock(content string) (string, bool) {
	bi, ei, ok := blockBounds(content)
	if !ok {
		return content, false
	}
	before := strings.TrimRight(content[:bi], "\n")
	after := strings.TrimLeft(content[ei:], "\n")
	var sb strings.Builder
	sb.WriteString(before)
	if before != "" && after != "" {
		sb.WriteString("\n\n")
	}
	sb.WriteString(after)
	next := sb.String()
	if strings.TrimSpace(next) != "" && !strings.HasSuffix(next, "\n") {
		next += "\n"
	}
	return next, true
}

func claudeAbs(dir string) string {
	return filepath.Join(dir, ".claude", "skills", "seo", "SKILL.md")
}

func agentsAbs(dir string) string {
	return filepath.Join(dir, agentsRel)
}

func hasClaude(dir string) bool {
	return dirExists(filepath.Join(dir, ".claude"))
}

func hasCodex(dir string) bool {
	return fileExists(agentsAbs(dir)) || dirExists(filepath.Join(dir, ".codex"))
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func readFileOrEmpty(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}
