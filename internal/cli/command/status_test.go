package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCommand(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "active-skill")
	createSkill(t, activeDir, "conflict")
	createSkill(t, disabledDir, "disabled-skill")
	createSkill(t, disabledDir, "conflict")
	if err := os.MkdirAll(filepath.Join(activeDir, "invalid-skill"), 0o755); err != nil {
		t.Fatalf("create invalid skill: %v", err)
	}

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewStatus()
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute status command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 output lines, got %d:\n%s", len(lines), output.String())
	}

	assertFields(t, lines[0], "Installed", "4")
	assertFields(t, lines[1], "Active", "1")
	assertFields(t, lines[2], "Disabled", "1")
	assertFields(t, lines[3], "Conflict", "1")
	assertFields(t, lines[4], "Broken", "0")
	assertFields(t, lines[5], "Invalid", "1")
	assertFields(t, lines[6], "Active", "dir", activeDir)
	assertFields(t, lines[7], "Disabled", "dir", disabledDir)
}
