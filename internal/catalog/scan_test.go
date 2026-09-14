package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createTestSkill(t, activeDir, "research")
	createTestSkill(t, activeDir, "code-review")

	createTestSkill(t, disabledDir, "prototype")
	createTestSkill(t, disabledDir, "research")

	invalidDir := filepath.Join(activeDir, "not-a-skill")

	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("create invalid directory: %v", err)
	}

	skills, err := Scan(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}

	if len(skills) != 4 {
		t.Fatalf("expected 4 skills, got %d", len(skills))
	}

	assertSkillState(t, skills[0], "code-review", StateActive)
	assertSkillState(t, skills[1], "not-a-skill", StateInvalid)
	assertSkillState(t, skills[2], "prototype", StateDisabled)
	assertSkillState(t, skills[3], "research", StateConflict)

	if got, want := skills[1].ActiveIssue, "missing SKILL.md"; got != want {
		t.Fatalf("invalid skill issue: got %q, want %q", got, want)
	}
}

func TestScanDetectsInstallerRecreatedActiveCopy(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createTestSkill(t, disabledDir, "research")
	createTestSkill(t, activeDir, "research")

	skills, err := Scan(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills: got %d, want 1", len(skills))
	}
	if skills[0].State != StateConflict {
		t.Fatalf("state: got %s, want conflict", skills[0].State)
	}
	if skills[0].ActivePath != filepath.Join(activeDir, "research") {
		t.Fatalf("active path: %q", skills[0].ActivePath)
	}
	if skills[0].DisabledPath != filepath.Join(disabledDir, "research") {
		t.Fatalf("disabled path: %q", skills[0].DisabledPath)
	}
	if _, err := os.Stat(skills[0].ActiveSkillFile); err != nil {
		t.Fatalf("active copy was removed: %v", err)
	}
	if _, err := os.Stat(skills[0].DisabledSkillFile); err != nil {
		t.Fatalf("disabled copy was removed: %v", err)
	}
}

func TestScanBrokenAndSymlinkSkills(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	targetDir := filepath.Join(root, "target")

	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}

	createTestSkill(t, targetDir, "source")

	linkPath := filepath.Join(activeDir, "linked")
	if err := os.Symlink(filepath.Join(targetDir, "source"), linkPath); err != nil {
		if os.PathSeparator == '\\' {
			t.Skipf("directory symlinks are unavailable: %v", err)
		}
		t.Fatalf("create directory symlink: %v", err)
	}

	brokenPath := filepath.Join(disabledDir, "broken")
	if err := os.Symlink(filepath.Join(root, "missing"), brokenPath); err != nil {
		if os.PathSeparator == '\\' {
			t.Skipf("directory symlinks are unavailable: %v", err)
		}
		t.Fatalf("create broken symlink: %v", err)
	}

	skills, err := Scan(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	assertSkillState(t, skills[0], "broken", StateBroken)
	assertSkillState(t, skills[1], "linked", StateActive)

	if skills[1].ActiveSource != SourceSymlink {
		t.Fatalf("expected symlink source, got %s", skills[1].ActiveSource)
	}

	if got, want := skills[0].DisabledIssue, "link target does not exist"; got != want {
		t.Fatalf("broken skill issue: got %q, want %q", got, want)
	}
}

func TestScanMissingDirectories(t *testing.T) {
	root := t.TempDir()

	skills, err := Scan(
		filepath.Join(root, "missing-active"),
		filepath.Join(root, "missing-disabled"),
	)
	if err != nil {
		t.Fatalf("scan missing directories: %v", err)
	}

	if len(skills) != 0 {
		t.Fatalf("expected no skills, got %d", len(skills))
	}
}

func TestScanSkillMetadata(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	skillDir := filepath.Join(activeDir, "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: Code Review\ndescription: Review code carefully.\n---\n# Content\n"),
		0o644,
	); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	skills, err := Scan(activeDir, filepath.Join(root, "disabled"))
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills: got %d, want 1", len(skills))
	}

	if skills[0].Name != "Code Review" {
		t.Fatalf("name: got %q, want %q", skills[0].Name, "Code Review")
	}
	if skills[0].Description != "Review code carefully." {
		t.Fatalf("description: got %q, want %q", skills[0].Description, "Review code carefully.")
	}
	if skills[0].SkillFile != filepath.Join(skillDir, "SKILL.md") {
		t.Fatalf("skill file: got %q", skills[0].SkillFile)
	}
}

func createTestSkill(
	t *testing.T,
	parent string,
	name string,
) {
	t.Helper()

	dir := filepath.Join(parent, name)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}

	skillFile := filepath.Join(dir, "SKILL.md")

	if err := os.WriteFile(
		skillFile,
		[]byte("# Test Skill\n"),
		0o644,
	); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func assertSkillState(
	t *testing.T,
	skill Skill,
	wantID string,
	wantState State,
) {
	t.Helper()

	if skill.ID != wantID {
		t.Errorf(
			"expected skill ID %q, got %q",
			wantID,
			skill.ID,
		)
	}

	if skill.State != wantState {
		t.Errorf(
			"expected state %q, got %q",
			wantState,
			skill.State,
		)
	}
}
