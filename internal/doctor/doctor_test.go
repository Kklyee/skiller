package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kklyee/skiller/internal/paths"
)

func TestInspectHealthy(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}

	createSkill(t, activeDir, "active")
	createSkill(t, disabledDir, "disabled")

	report := Inspect(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Journal:  filepath.Join(root, "transaction.json"),
	})

	if report.Overall != Healthy {
		t.Fatalf("overall: got %s, want healthy", report.Overall)
	}
	if report.Summary.Active != 1 || report.Summary.Disabled != 1 {
		t.Fatalf("summary: got %+v", report.Summary)
	}
}

func TestInspectWarningsAndErrors(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}

	createSkill(t, activeDir, "conflict")
	if err := os.MkdirAll(filepath.Join(activeDir, "invalid"), 0o755); err != nil {
		t.Fatalf("create invalid directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "transaction.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}
	createSkill(t, disabledDir, "conflict")

	report := Inspect(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Journal:  filepath.Join(root, "transaction.json"),
	})

	if report.Overall != Error {
		t.Fatalf("overall: got %s, want error", report.Overall)
	}
	if report.Summary.Conflict != 1 || report.Summary.Invalid != 1 {
		t.Fatalf("summary: got %+v", report.Summary)
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
