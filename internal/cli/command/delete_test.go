package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/pin"
)

func TestDeleteCommandListsInstalledSkillsAndDeletesMultipleSelections(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	pinsPath := filepath.Join(root, "pins.toml")
	setDeletePaths(t, root, activeDir, disabledDir, pinsPath)
	createSkill(t, activeDir, "alpha")
	createSkill(t, disabledDir, "beta")

	var output bytes.Buffer
	cmd := NewDelete()
	cmd.SetIn(strings.NewReader("1,2\ny\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute delete command: %v", err)
	}

	for _, want := range []string{
		"Installed skills:",
		"1) alpha [active]",
		"2) beta [disabled]",
		"Deleted 2 skills",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("delete output missing %q:\n%s", want, output.String())
		}
	}
	for _, path := range []string{
		filepath.Join(activeDir, "alpha"),
		filepath.Join(disabledDir, "beta"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("deleted path %s still exists: %v", path, err)
		}
	}
}

func TestDeleteCommandMarksPinnedSkillInPicker(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	pinsPath := filepath.Join(root, "pins.toml")
	setDeletePaths(t, root, activeDir, filepath.Join(root, "disabled"), pinsPath)
	createSkill(t, activeDir, "alpha")

	if _, err := pin.New(pinsPath).Add("alpha"); err != nil {
		t.Fatalf("pin skill: %v", err)
	}

	var output bytes.Buffer
	cmd := NewDelete()
	cmd.SetIn(strings.NewReader("1\ny\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute delete pinned skill: %v", err)
	}
	if !strings.Contains(output.String(), "1) alpha [active, pinned]") {
		t.Fatalf("pinned status missing from picker:\n%s", output.String())
	}
	remaining, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("read pins after deletion: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("pins after deletion = %v", remaining)
	}
}

func TestDeleteCommandRejectsUnknownArgumentAfterListingInstalledSkills(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	setDeletePaths(t, root, activeDir, filepath.Join(root, "disabled"), filepath.Join(root, "pins.toml"))
	createSkill(t, activeDir, "alpha")

	var output bytes.Buffer
	cmd := NewDelete()
	cmd.SetArgs([]string{"--yes", "missing"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `skill "missing" is not installed`) {
		t.Fatalf("unknown delete error = %v", err)
	}
	if !strings.Contains(output.String(), "1) alpha [active]") {
		t.Fatalf("installed list missing for rejected argument:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "alpha")); err != nil {
		t.Fatalf("known skill changed after rejected delete: %v", err)
	}
}

func TestDeleteCommandCanCancelPermanentDeletion(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	setDeletePaths(t, root, activeDir, filepath.Join(root, "disabled"), filepath.Join(root, "pins.toml"))
	createSkill(t, activeDir, "alpha")

	var output bytes.Buffer
	cmd := NewDelete()
	cmd.SetArgs([]string{"alpha"})
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute cancelled delete: %v", err)
	}
	if !strings.Contains(output.String(), "Cancelled") {
		t.Fatalf("cancel output missing:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "alpha")); err != nil {
		t.Fatalf("skill removed after cancellation: %v", err)
	}
}

func setDeletePaths(t *testing.T, root, active, disabled, pins string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_GROUPS_DIR", filepath.Join(root, "groups"))
	t.Setenv("SKILLER_PROFILES_DIR", filepath.Join(root, "profiles"))
	t.Setenv("SKILLER_PINS_FILE", pins)
	t.Setenv("SKILLER_PROVENANCE_FILE", filepath.Join(root, "provenance.toml"))
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(root, "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(root, "lock"))
}
