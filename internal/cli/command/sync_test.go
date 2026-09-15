package command

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/project"
)

func TestSyncDirectSkills(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	createSkill(t, disabledDir, "pinned")
	writeProjectConfig(t, projectDir, "skills = [\"wanted\"]\n")
	setSyncPaths(t, activeDir, disabledDir, filepath.Join(root, "groups"), filepath.Join(root, "profiles"))
	if _, err := pin.New(filepath.Join(root, "pins.toml")).Add("pinned"); err != nil {
		t.Fatalf("pin skill: %v", err)
	}
	t.Chdir(projectDir)

	var output bytes.Buffer
	cmd := NewSync()
	cmd.SetArgs([]string{"--dry-run"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync dry run: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "  + wanted") || !strings.Contains(output.String(), "  + pinned") || !strings.Contains(output.String(), "  - old") {
		t.Fatalf("dry-run output: %q", output.String())
	}
	for _, want := range []string{"Project config:", project.ConfigFileName, "Target: project", "Enable (2)", "Disable (1)", "Keep (0)"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("dry-run context or section missing %q:\n%s", want, output.String())
		}
	}
	assertProfilePathExists(t, filepath.Join(activeDir, "old", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(disabledDir, "wanted", "SKILL.md"))

	output.Reset()
	cmd = NewSync()
	cmd.SetArgs(nil)
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Synced ") || !strings.Contains(output.String(), project.ConfigFileName) {
		t.Fatalf("sync output: %q", output.String())
	}
	assertProfilePathExists(t, filepath.Join(disabledDir, "old", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(activeDir, "wanted", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(activeDir, "pinned", "SKILL.md"))
}

func TestSyncProfile(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	groups := group.New(groupsDir)
	if _, err := groups.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groups.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	profiles := profile.New(profilesDir)
	if _, err := profiles.Create("go-backend"); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profiles.Update("go-backend", profile.Profile{Name: "go-backend", Groups: []string{"coding"}}); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	writeProjectConfig(t, projectDir, "profile = \"go-backend\"\n")
	setSyncPaths(t, activeDir, disabledDir, groupsDir, profilesDir)
	t.Chdir(projectDir)

	var output bytes.Buffer
	cmd := NewSync()
	cmd.SetArgs([]string{"--dry-run"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync profile dry run: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Target: go-backend") {
		t.Fatalf("profile target missing:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "  + wanted") || !strings.Contains(output.String(), "  - old") {
		t.Fatalf("profile dry-run output: %q", output.String())
	}
}

func TestSyncProjectOverridesProfile(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	createSkill(t, activeDir, "base")
	createSkill(t, activeDir, "excluded")
	createSkill(t, disabledDir, "included")
	createSkill(t, disabledDir, "old")

	groups := group.New(groupsDir)
	if _, err := groups.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groups.Add("coding", "base"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	profiles := profile.New(profilesDir)
	if _, err := profiles.Create("backend"); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profiles.Update("backend", profile.Profile{Name: "backend", Groups: []string{"coding"}}); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	writeProjectConfig(t, projectDir, "profile = \"backend\"\ninclude = [\"included\"]\nexclude = [\"excluded\"]\n")
	setSyncPaths(t, activeDir, disabledDir, groupsDir, profilesDir)
	t.Chdir(projectDir)

	var output bytes.Buffer
	cmd := NewSync()
	cmd.SetArgs([]string{"--dry-run"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync project overrides: %v\n%s", err, output.String())
	}
	text := output.String()
	if !strings.Contains(text, "  + included") || !strings.Contains(text, "  - excluded") || !strings.Contains(text, "  = base") {
		t.Fatalf("override plan: %q", text)
	}
	for _, want := range []string{"active (1)", "disabled (1)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("keep breakdown missing %q: %q", want, text)
		}
	}
}

func TestSyncRequiresProjectConfig(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	setSyncPaths(t, filepath.Join(root, "active"), filepath.Join(root, "disabled"), filepath.Join(root, "groups"), filepath.Join(root, "profiles"))
	t.Chdir(projectDir)

	cmd := NewSync()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), project.ConfigFileName) {
		t.Fatalf("expected project config error, got %v", err)
	}
}

func TestSyncCheckReportsOutOfSyncWithDedicatedExitCode(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	writeProjectConfig(t, projectDir, "skills = [\"wanted\"]\n")
	setSyncPaths(t, activeDir, disabledDir, filepath.Join(root, "groups"), filepath.Join(root, "profiles"))
	t.Chdir(projectDir)

	var output bytes.Buffer
	cmd := NewSync()
	cmd.SetArgs([]string{"--check"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "out of sync") {
		t.Fatalf("sync check error: %v\n%s", err, output.String())
	}
	var coded interface{ ExitCode() int }
	if !errors.As(err, &coded) || coded.ExitCode() != 2 {
		t.Fatalf("sync check exit code: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("sync check changed old skill: %v", err)
	}
}

func writeProjectConfig(t *testing.T, dir, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, project.ConfigFileName), []byte(data), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func setSyncPaths(t *testing.T, active, disabled, groups, profiles string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_GROUPS_DIR", groups)
	t.Setenv("SKILLER_PROFILES_DIR", profiles)
	t.Setenv("SKILLER_PINS_FILE", filepath.Join(filepath.Dir(disabled), "pins.toml"))
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(filepath.Dir(disabled), "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(filepath.Dir(disabled), "lock"))
}
