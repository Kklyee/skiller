package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/group"
)

func TestUseDryRun(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	setUsePaths(t, activeDir, disabledDir, groupsDir)

	var output bytes.Buffer
	command := NewUse()
	command.SetArgs([]string{"coding", "--dry-run"})
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute dry run: %v", err)
	}
	if !strings.Contains(output.String(), "  + wanted") || !strings.Contains(output.String(), "  - old") {
		t.Fatalf("unexpected plan:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "wanted")); err != nil {
		t.Fatalf("dry run changed disabled skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old")); err != nil {
		t.Fatalf("dry run changed active skill: %v", err)
	}
}

func TestUseConfirmsAndApplies(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	setUsePaths(t, activeDir, disabledDir, groupsDir)

	var output bytes.Buffer
	command := NewUse()
	command.SetArgs([]string{"coding"})
	command.SetIn(strings.NewReader("y\n"))
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute use: %v", err)
	}
	if !strings.Contains(output.String(), "Applied group coding") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not enabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old was not disabled: %v", err)
	}
}

func TestUseCancelsWithoutChanges(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	setUsePaths(t, activeDir, disabledDir, groupsDir)

	var output bytes.Buffer
	command := NewUse()
	command.SetArgs([]string{"coding"})
	command.SetIn(strings.NewReader("n\n"))
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute use: %v", err)
	}
	if !strings.Contains(output.String(), "Cancelled") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old")); err != nil {
		t.Fatalf("cancel changed active skill: %v", err)
	}
}

func setUsePaths(t *testing.T, active, disabled, groups string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_GROUPS_DIR", groups)
}
