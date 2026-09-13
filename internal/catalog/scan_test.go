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

	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}

	assertSkillState(t, skills[0], "code-review", StateActive)
	assertSkillState(t, skills[1], "prototype", StateDisabled)
	assertSkillState(t, skills[2], "research", StateConflict)
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
