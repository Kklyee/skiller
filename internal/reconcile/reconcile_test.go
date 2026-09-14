package reconcile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
)

func TestBuildPlan(t *testing.T) {
	plan := Build(
		group.Group{Name: "coding", Skills: []string{"wanted", "needs-enable", "missing"}},
		[]catalog.Skill{
			{ID: "wanted", State: catalog.StateActive},
			{ID: "needs-enable", State: catalog.StateDisabled},
			{ID: "old", State: catalog.StateActive},
			{ID: "ignored", State: catalog.StateDisabled},
			{ID: "conflict", State: catalog.StateConflict},
		},
	)

	assertStrings(t, plan.Enable, "needs-enable")
	assertStrings(t, plan.Disable, "old")
	assertStrings(t, plan.Keep, "ignored", "wanted")
	assertStrings(t, plan.Missing, "missing")
	if len(plan.Issues) != 1 || plan.Issues[0] != "keep conflict: conflict" {
		t.Fatalf("issues: got %v", plan.Issues)
	}
	if plan.Changes() != 2 {
		t.Fatalf("changes: got %d, want 2", plan.Changes())
	}
	if !plan.HasIssues() {
		t.Fatal("expected plan issues")
	}
}

func TestApplyPlan(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")

	plan := Plan{Enable: []string{"wanted"}, Disable: []string{"old"}}
	if err := Apply(activeDir, disabledDir, plan); err != nil {
		t.Fatalf("apply plan: %v", err)
	}

	if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not enabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old was not disabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("old active path still exists: %v", err)
	}
}

func createSkill(t *testing.T, parent, name string) {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func assertStrings(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("strings: got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("string %d: got %q, want %q", index, got[index], want[index])
		}
	}
}
