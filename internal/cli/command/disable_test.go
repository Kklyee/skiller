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

func TestDisableCommandMovesActiveSkill(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "research")
	createSkill(t, activeDir, "code-review")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewDisable()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("disable research: %v", err)
	}

	if got := strings.TrimSpace(output.String()); got != "Disabled research" {
		t.Fatalf(
			"unexpected output: got %q, want %q",
			got,
			"Disabled research",
		)
	}

	activePath := filepath.Join(activeDir, "research")

	if _, err := os.Lstat(activePath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf(
			"expected active skill to be removed, got error: %v",
			err,
		)
	}

	disabledSkillFile := filepath.Join(
		disabledDir,
		"research",
		"SKILL.md",
	)

	data, err := os.ReadFile(disabledSkillFile)
	if err != nil {
		t.Fatalf("read disabled SKILL.md: %v", err)
	}

	if got, want := string(data), "# Test Skill\n"; got != want {
		t.Fatalf(
			"unexpected SKILL.md contents: got %q, want %q",
			got,
			want,
		)
	}
}

func TestDisableCommandAlreadyDisabled(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewDisable()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("disable already-disabled skill: %v", err)
	}

	if got := strings.TrimSpace(output.String()); got != "research is already disabled" {
		t.Fatalf(
			"unexpected output: got %q, want %q",
			got,
			"research is already disabled",
		)
	}
}

func TestDisableCommandRejectsConflict(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	cmd := NewDisable()
	cmd.SetArgs([]string{"research"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}

	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf(
			"expected conflict error, got %q",
			err,
		)
	}

	assertPathExists(
		t,
		filepath.Join(activeDir, "research"),
	)

	assertPathExists(
		t,
		filepath.Join(disabledDir, "research"),
	)
}

func TestDisableCommandRejectsMissingSkill(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	cmd := NewDisable()
	cmd.SetArgs([]string{"research"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing skill error")
	}

	if !strings.Contains(
		err.Error(),
		`skill "research" is not installed`,
	) {
		t.Fatalf(
			"unexpected error: %q",
			err,
		)
	}
}

func assertPathExists(
	t *testing.T,
	path string,
) {
	t.Helper()

	if _, err := os.Lstat(path); err != nil {
		t.Fatalf(
			"expected path %q to exist: %v",
			path,
			err,
		)
	}
}
