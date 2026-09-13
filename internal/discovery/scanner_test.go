package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kklyee/skiller/internal/domain"
)

func TestScannerScan(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	mustCreateSkill(t, activeDir, "research")
	mustCreateSkill(t, activeDir, "code-review")
	mustCreateSkill(t, disabledDir, "prototype")

	invalidDir := filepath.Join(activeDir, "not-a-skill")

	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("create invalid directory: %v", err)
	}

	scanner := NewScanner()

	skills, err := scanner.Scan(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}

	if len(skills) != 3 {
		t.Fatalf(
			"expected 3 skills, got %d",
			len(skills),
		)
	}

	assertSkill(
		t,
		skills[0],
		"code-review",
		domain.SkillStateActive,
	)

	assertSkill(
		t,
		skills[1],
		"prototype",
		domain.SkillStateDisabled,
	)

	assertSkill(
		t,
		skills[2],
		"research",
		domain.SkillStateActive,
	)
}

func mustCreateSkill(
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
		[]byte("# test skill"),
		0o644,
	); err != nil {
		t.Fatalf("create SKILL.md: %v", err)
	}
}

func assertSkill(
	t *testing.T,
	skill domain.Skill,
	wantID string,
	wantState domain.SkillState,
) {
	t.Helper()

	if skill.ID != wantID {
		t.Errorf(
			"expected ID %q, got %q",
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
