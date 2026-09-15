package command

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
)

func TestInstallCommandBridgesSkillsCLIAndRecordsProvenance(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	provenancePath := filepath.Join(root, "provenance.toml")
	setInstallerPaths(t, activeDir, disabledDir, provenancePath)

	var gotArgs []string
	run := func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		gotArgs = append([]string(nil), args...)
		createSkill(t, activeDir, "code-review")
		return nil
	}

	cmd := newInstallerCommand("install", "add", 1, run)
	cmd.SetArgs([]string{"owner/repo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("install skill: %v", err)
	}
	if strings.Join(gotArgs, " ") != "skills add owner/repo" {
		t.Fatalf("installer args: %q", gotArgs)
	}
	entry, ok, err := skillprovenance.New(provenancePath).Get("code-review")
	if err != nil {
		t.Fatalf("read installed provenance: %v", err)
	}
	if !ok || entry.Source != "owner/repo" || entry.Repository != "owner/repo" || entry.Installer != "npx skills" {
		t.Fatalf("installed provenance: got %#v, %t", entry, ok)
	}
}

func TestUpdateCommandKeepsDisabledSkillDisabled(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	setInstallerPaths(t, activeDir, disabledDir, filepath.Join(root, "provenance.toml"))
	createSkill(t, activeDir, "always-active")
	createSkill(t, disabledDir, "research")

	seenActiveDuringUpdate := false
	run := func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		if _, err := os.Stat(filepath.Join(activeDir, "research", "SKILL.md")); err == nil {
			seenActiveDuringUpdate = true
		}
		return nil
	}

	cmd := newInstallerCommand("update", "update", 0, run)
	cmd.SetArgs([]string{"research"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update skill: %v", err)
	}
	if !seenActiveDuringUpdate {
		t.Fatal("disabled skill was not available to installer during update")
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("disabled skill was not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "research", "SKILL.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("research remained active, stat error: %v", err)
	}
}

func TestUpdateCommandRestoresDisabledSkillWhenInstallerFails(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	setInstallerPaths(t, activeDir, disabledDir, filepath.Join(root, "provenance.toml"))
	createSkill(t, disabledDir, "research")

	cmd := newInstallerCommand("update", "update", 0, func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		return errors.New("skills CLI failed")
	})
	cmd.SetArgs([]string{"research"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "skills CLI failed") {
		t.Fatalf("installer error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("disabled skill was not restored after failure: %v", err)
	}
}

func TestInstallCommandReportsConflictWithoutRemovingCopies(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	setInstallerPaths(t, activeDir, disabledDir, filepath.Join(root, "provenance.toml"))
	createSkill(t, disabledDir, "research")

	cmd := newInstallerCommand("install", "add", 1, func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		createSkill(t, activeDir, "research")
		return nil
	})
	cmd.SetArgs([]string{"owner/repo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflict error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("active copy was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("disabled copy was removed: %v", err)
	}
}

func TestRemoveCommandDeletesProvenanceForRemovedSkill(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	provenancePath := filepath.Join(root, "provenance.toml")
	setInstallerPaths(t, activeDir, filepath.Join(root, "disabled"), provenancePath)
	createSkill(t, activeDir, "research")
	if err := skillprovenance.New(provenancePath).Set("research", skillprovenance.Entry{Source: "owner/repo"}); err != nil {
		t.Fatalf("set provenance: %v", err)
	}

	cmd := newInstallerCommand("remove", "remove", 1, func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		return os.RemoveAll(filepath.Join(activeDir, "research"))
	})
	cmd.SetArgs([]string{"research"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("remove skill: %v", err)
	}
	if _, ok, err := skillprovenance.New(provenancePath).Get("research"); err != nil || ok {
		t.Fatalf("removed provenance: ok=%t err=%v", ok, err)
	}
}

func setInstallerPaths(t *testing.T, active, disabled, provenance string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_PROVENANCE_FILE", provenance)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(filepath.Dir(disabled), "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(filepath.Dir(disabled), "lock"))
}
