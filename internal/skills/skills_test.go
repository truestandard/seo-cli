package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestandard/seo-cli/skill"
)

func project(t *testing.T, with ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, p := range with {
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(filepath.Join(dir, p), 0o755); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(filepath.Join(dir, p), []byte("# Existing\n\nkeep me\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNothingDetectedWritesNothing(t *testing.T) {
	dir := project(t)
	results, err := Install(dir)
	if err != nil || len(results) != 0 {
		t.Fatalf("got %v %v", results, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Fatal(".claude must not be created")
	}
}

func TestClaudeInstallIsIdempotent(t *testing.T) {
	dir := project(t, ".claude/")
	first, _ := Install(dir)
	second, _ := Install(dir)
	if first[0].Action != ActionInstalled || second[0].Action != ActionUnchanged {
		t.Fatalf("got %v then %v", first, second)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "skills", "seo", "SKILL.md"))
	if string(data) != skill.ClaudeSkill() {
		t.Fatal("skill content differs")
	}
	if !strings.HasPrefix(string(data), "---\nname: seo\n") {
		t.Fatal("frontmatter missing")
	}
}

func TestAgentsBlockPreservesSurroundingContent(t *testing.T) {
	dir := project(t, "AGENTS.md")
	if r, _ := Install(dir); r[0].Action != ActionInstalled {
		t.Fatalf("got %v", r)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	text := string(data)
	if !strings.HasPrefix(text, "# Existing\n\nkeep me\n\n"+skill.BeginPrefix) || !strings.HasSuffix(text, skill.EndMarker+"\n") {
		t.Fatalf("got:\n%s", text)
	}
	if r, _ := Install(dir); r[0].Action != ActionUnchanged {
		t.Fatalf("second install: %v", r)
	}
}

func TestCodexDirCreatesAgentsFile(t *testing.T) {
	dir := project(t, ".codex/")
	if r, _ := Install(dir); r[0].Action != ActionInstalled {
		t.Fatalf("got %v", r)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if string(data) != skill.AgentsBlock() {
		t.Fatal("file should hold only the block")
	}
}

func TestTruncatedBlockIsStaleThenFixed(t *testing.T) {
	dir := project(t, "AGENTS.md")
	Install(dir)
	path := filepath.Join(dir, "AGENTS.md")
	data, _ := os.ReadFile(path)
	truncated := string(data)[:len(data)-len(skill.EndMarker)-1] + "\n"
	os.WriteFile(path, []byte(truncated), 0o644)
	if s := List(dir); !s[0].Installed || !s[0].Stale {
		t.Fatalf("got %+v", s)
	}
	report, _ := Doctor(dir, true)
	if len(report.Fixed) != 1 || report.Targets[0].Stale {
		t.Fatalf("got %+v", report)
	}
	fixed, _ := os.ReadFile(path)
	if strings.Count(string(fixed), skill.BeginPrefix) != 1 {
		t.Fatal("block duplicated")
	}
}

func TestUninstallRemovesOnlyOurs(t *testing.T) {
	dir := project(t, ".claude/", "AGENTS.md")
	Install(dir)
	results, _ := Uninstall(dir)
	if results[0].Action != ActionRemoved || results[1].Action != ActionRemoved {
		t.Fatalf("got %v", results)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if string(data) != "# Existing\n\nkeep me\n" {
		t.Fatalf("got %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "seo")); !os.IsNotExist(err) {
		t.Fatal("claude skill dir remains")
	}
}

func TestUninstallDeletesAgentsFileThatHeldOnlyTheBlock(t *testing.T) {
	dir := project(t, ".codex/")
	Install(dir)
	Uninstall(dir)
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("AGENTS.md should be deleted")
	}
}
