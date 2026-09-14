package command

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableCommandMovesDisabledSkill(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewEnable()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("enable research: %v", err)
	}

	if got := strings.TrimSpace(output.String()); got != "Enabled research" {
		t.Fatalf(
			"unexpected output: got %q, want %q",
			got,
			"Enabled research",
		)
	}

	if _, err := os.Lstat(filepath.Join(disabledDir, "research")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected disabled skill to be removed, got error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(activeDir, "research", "SKILL.md"))
	if err != nil {
		t.Fatalf("read active SKILL.md: %v", err)
	}

	if got, want := string(data), "# Test Skill\n"; got != want {
		t.Fatalf("unexpected SKILL.md contents: got %q, want %q", got, want)
	}
}

func TestEnableCommandAlreadyEnabled(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewEnable()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("enable already-enabled skill: %v", err)
	}

	if got := strings.TrimSpace(output.String()); got != "research is already enabled" {
		t.Fatalf(
			"unexpected output: got %q, want %q",
			got,
			"research is already enabled",
		)
	}
}

func TestEnableCommandRejectsConflict(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	cmd := NewEnable()
	cmd.SetArgs([]string{"research"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got %q", err)
	}

	assertPathExists(t, filepath.Join(activeDir, "research"))
	assertPathExists(t, filepath.Join(disabledDir, "research"))
}

func TestEnableCommandRejectsExistingDestination(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, disabledDir, "research")
	createSkill(t, activeDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	cmd := NewEnable()
	cmd.SetArgs([]string{"research"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got %q", err)
	}
}
