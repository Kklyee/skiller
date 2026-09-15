package command

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/bundle"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
)

func TestExportCommandWritesEnvironmentBundle(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	pinsPath := filepath.Join(root, "pins.toml")
	provenancePath := filepath.Join(root, "provenance.toml")
	setBundlePaths(t, activeDir, disabledDir, groupsDir, profilesDir, pinsPath, provenancePath)
	createSkill(t, activeDir, "code-review")
	createSkill(t, disabledDir, "research")
	groupStore := group.New(groupsDir)
	if _, err := groupStore.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groupStore.Add("coding", "code-review"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	profileStore := profile.New(profilesDir)
	if _, err := profileStore.Create("backend"); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	storedProfile, err := profileStore.Get("backend")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	storedProfile.Groups = []string{"coding"}
	if err := profileStore.Update("backend", storedProfile); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if err := pin.New(pinsPath).Replace([]string{"code-review"}); err != nil {
		t.Fatalf("write pins: %v", err)
	}
	if err := skillprovenance.New(provenancePath).Set("code-review", skillprovenance.Entry{Source: "owner/repo"}); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	path := filepath.Join(root, "environment.toml")
	var output bytes.Buffer
	cmd := NewExport()
	cmd.SetArgs([]string{path})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("export environment: %v", err)
	}
	got, err := bundle.Load(path)
	if err != nil {
		t.Fatalf("load exported bundle: %v", err)
	}
	if len(got.Active) != 1 || got.Active[0] != "code-review" || len(got.Pins) != 1 || len(got.Groups) != 1 || len(got.Profiles) != 1 || got.Provenance["code-review"].Source != "owner/repo" {
		t.Fatalf("exported bundle: %#v", got)
	}
}

func TestImportCommandRestoresEnvironmentAndMetadata(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	pinsPath := filepath.Join(root, "pins.toml")
	provenancePath := filepath.Join(root, "provenance.toml")
	setBundlePaths(t, activeDir, disabledDir, groupsDir, profilesDir, pinsPath, provenancePath)
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "code-review")
	bundlePath := filepath.Join(root, "environment.toml")
	if err := bundle.Write(bundlePath, bundle.Bundle{
		Version:  bundle.Version,
		Active:   []string{"code-review"},
		Pins:     []string{"code-review"},
		Groups:   []group.Group{{Name: "coding", Skills: []string{"code-review"}}},
		Profiles: []profile.Profile{{Name: "backend", Groups: []string{"coding"}}},
		Provenance: map[string]skillprovenance.Entry{
			"code-review": {Source: "owner/repo", Repository: "owner/repo", Installer: "npx skills"},
		},
	}); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	var output bytes.Buffer
	cmd := NewImport()
	cmd.SetArgs([]string{bundlePath, "--yes"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import environment: %v\n%s", err, output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "code-review", "SKILL.md")); err != nil {
		t.Fatalf("imported skill not active: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old skill not disabled: %v", err)
	}
	if _, err := group.New(groupsDir).Get("coding"); err != nil {
		t.Fatalf("imported group: %v", err)
	}
	if _, err := profile.New(profilesDir).Get("backend"); err != nil {
		t.Fatalf("imported profile: %v", err)
	}
	if got, err := pin.New(pinsPath).List(); err != nil || len(got) != 1 || got[0] != "code-review" {
		t.Fatalf("imported pins: %v, %v", got, err)
	}
}

func TestImportCommandRejectsMissingSkillsWithoutInstallFlag(t *testing.T) {
	root := t.TempDir()
	setBundlePaths(t, filepath.Join(root, "active"), filepath.Join(root, "disabled"), filepath.Join(root, "groups"), filepath.Join(root, "profiles"), filepath.Join(root, "pins.toml"), filepath.Join(root, "provenance.toml"))
	bundlePath := filepath.Join(root, "environment.toml")
	if err := bundle.Write(bundlePath, bundle.Bundle{Version: bundle.Version, Active: []string{"missing"}}); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	cmd := NewImport()
	cmd.SetArgs([]string{bundlePath})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing skills") {
		t.Fatalf("missing skill error: %v", err)
	}
}

func TestImportCommandInstallsMissingSkillFromProvenance(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	setBundlePaths(t, activeDir, disabledDir, filepath.Join(root, "groups"), filepath.Join(root, "profiles"), filepath.Join(root, "pins.toml"), filepath.Join(root, "provenance.toml"))
	bundlePath := filepath.Join(root, "environment.toml")
	if err := bundle.Write(bundlePath, bundle.Bundle{
		Version: bundle.Version,
		Active:  []string{"missing"},
		Provenance: map[string]skillprovenance.Entry{
			"missing": {Repository: "owner/repo"},
		},
	}); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	var gotArgs []string
	run := func(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
		gotArgs = append([]string(nil), args...)
		createSkill(t, activeDir, "missing")
		return nil
	}
	cmd := newImportCommand(run)
	cmd.SetArgs([]string{bundlePath, "--install-missing", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("import missing skill: %v", err)
	}
	if strings.Join(gotArgs, " ") != "skills add owner/repo --skill missing -y" {
		t.Fatalf("install args: %q", gotArgs)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "missing", "SKILL.md")); err != nil {
		t.Fatalf("missing skill not installed: %v", err)
	}
}

func setBundlePaths(t *testing.T, active, disabled, groups, profiles, pinsPath, provenancePath string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_GROUPS_DIR", groups)
	t.Setenv("SKILLER_PROFILES_DIR", profiles)
	t.Setenv("SKILLER_PINS_FILE", pinsPath)
	t.Setenv("SKILLER_PROVENANCE_FILE", provenancePath)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(filepath.Dir(disabled), "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(filepath.Dir(disabled), "lock"))
}
