package transaction

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
)

func TestApplyRecordsAndRemovesJournal(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	pathSet := testPaths(root, activeDir, disabledDir)

	if err := Apply(pathSet, reconcile.Plan{
		Group:   "coding",
		Enable:  []string{"wanted"},
		Disable: []string{"old"},
	}); err != nil {
		t.Fatalf("apply transaction: %v", err)
	}

	if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not enabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old was not disabled: %v", err)
	}
	if _, err := os.Lstat(pathSet.Journal); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("journal still exists: %v", err)
	}
	if _, err := os.Lstat(pathSet.Lock); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("lock still exists: %v", err)
	}
}

func TestApplyRollsBackCompletedMoves(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, disabledDir, "wanted")
	pathSet := testPaths(root, activeDir, disabledDir)

	err := Apply(pathSet, reconcile.Plan{
		Group:   "coding",
		Enable:  []string{"wanted"},
		Disable: []string{"missing"},
	})
	if err == nil {
		t.Fatal("expected transaction failure")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unexpected transaction error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not rolled back: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "wanted")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("wanted active path remains: %v", err)
	}
	if _, err := os.Lstat(pathSet.Journal); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("rollback journal still exists: %v", err)
	}
}

func TestApplyRejectsExistingJournal(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, disabledDir, "wanted")
	pathSet := testPaths(root, activeDir, disabledDir)
	if err := os.WriteFile(pathSet.Journal, []byte("unfinished"), 0o600); err != nil {
		t.Fatalf("create journal: %v", err)
	}

	err := Apply(pathSet, reconcile.Plan{Enable: []string{"wanted"}})
	if err == nil || !strings.Contains(err.Error(), "unfinished transaction journal") {
		t.Fatalf("unexpected journal error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "wanted")); err != nil {
		t.Fatalf("existing journal changed skill state: %v", err)
	}
}

func testPaths(root, active, disabled string) paths.Set {
	return paths.Set{
		Active:   active,
		Disabled: disabled,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
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
