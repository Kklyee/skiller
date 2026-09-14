package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
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

func TestInspectReportsInvalidPins(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	pinsPath := filepath.Join(root, "pins.toml")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}
	if _, err := pin.New(pinsPath).Add("missing-skill"); err != nil {
		t.Fatalf("create pin: %v", err)
	}

	report := Inspect(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Pins:     pinsPath,
		Journal:  filepath.Join(root, "transaction.json"),
	})

	if report.Overall != Error {
		t.Fatalf("overall: got %s, want error", report.Overall)
	}
	if !hasCheckDetail(report, "Skills", "missing-skill") {
		t.Fatalf("invalid pin check missing from report: %+v", report.Checks)
	}
}

func hasCheckDetail(report Report, section, detail string) bool {
	for _, check := range report.Checks {
		if check.Section == section && strings.Contains(check.Detail, detail) {
			return true
		}
	}
	return false
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
