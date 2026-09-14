package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinListAndUnpinCommands(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	setPinPaths(t, activeDir, disabledDir, filepath.Join(root, "pins.toml"))

	var output bytes.Buffer
	cmd := NewPin()
	cmd.SetArgs([]string{"prototype", "missing-skill"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pin skills: %v", err)
	}
	if !strings.Contains(output.String(), "Pinned 2 skills") {
		t.Fatalf("pin output: %q", output.String())
	}

	output.Reset()
	cmd = NewPins()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if !strings.Contains(output.String(), "missing-skill") || !strings.Contains(output.String(), "missing") || !strings.Contains(output.String(), "prototype") || !strings.Contains(output.String(), "disabled") {
		t.Fatalf("pins output: %q", output.String())
	}

	output.Reset()
	cmd = NewUnpin()
	cmd.SetArgs([]string{"prototype"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unpin skill: %v", err)
	}
	if !strings.Contains(output.String(), "Unpinned 1 skills") {
		t.Fatalf("unpin output: %q", output.String())
	}
}

func setPinPaths(t *testing.T, active, disabled, pins string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_PINS_FILE", pins)
}
