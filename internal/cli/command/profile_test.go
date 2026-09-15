package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/environment"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/pin"
	"github.com/Kklyee/skiller/internal/profile"
)

func TestProfileCommandsAndUse(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	createSkill(t, disabledDir, "pinned")

	groups := group.New(groupsDir)
	if _, err := groups.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := groups.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}
	setProfilePaths(t, activeDir, disabledDir, groupsDir, profilesDir)
	if _, err := pin.New(filepath.Join(root, "pins.toml")).Add("pinned"); err != nil {
		t.Fatalf("pin skill: %v", err)
	}

	if output := executeProfileCommand(t, "create", "go-backend"); strings.TrimSpace(output) != "Created profile go-backend" {
		t.Fatalf("create output: %q", output)
	}

	output := executeProfileCommand(t, "edit", "go-backend", "--groups", "coding", "--skills", "wanted", "--exclude", "old")
	if strings.TrimSpace(output) != "Updated profile go-backend" {
		t.Fatalf("edit output: %q", output)
	}

	output = executeProfileCommand(t, "show", "go-backend")
	if !strings.Contains(output, "  coding") || !strings.Contains(output, "  wanted") || !strings.Contains(output, "  old") || !strings.Contains(output, "Missing groups:\n  none") {
		t.Fatalf("show output: %q", output)
	}

	output = executeProfileCommand(t, "list")
	if !strings.Contains(output, "PROFILE") || !strings.Contains(output, "go-backend") || !strings.Contains(output, "1       1       1") {
		t.Fatalf("list output: %q", output)
	}

	output = executeProfileCommand(t, "use", "go-backend", "--dry-run")
	if !strings.Contains(output, "  + wanted") || !strings.Contains(output, "  + pinned") || !strings.Contains(output, "  - old") {
		t.Fatalf("dry-run output: %q", output)
	}
	assertProfilePathExists(t, filepath.Join(activeDir, "old", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(disabledDir, "wanted", "SKILL.md"))

	var applied bytes.Buffer
	cmd := NewProfile()
	cmd.SetArgs([]string{"use", "go-backend"})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&applied)
	cmd.SetErr(&applied)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("use profile: %v\n%s", err, applied.String())
	}
	if !strings.Contains(applied.String(), "Applied profile go-backend") {
		t.Fatalf("apply output: %q", applied.String())
	}
	assertProfilePathExists(t, filepath.Join(disabledDir, "old", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(activeDir, "wanted", "SKILL.md"))
	assertProfilePathExists(t, filepath.Join(activeDir, "pinned", "SKILL.md"))
	gotTarget, ok, err := environment.New(filepath.Join(root, "state.toml")).Load()
	if err != nil {
		t.Fatalf("load applied profile: %v", err)
	}
	if !ok || gotTarget != (environment.Target{Kind: environment.KindProfile, Name: "go-backend"}) {
		t.Fatalf("applied profile target = %+v, loaded = %v", gotTarget, ok)
	}

	if output := executeProfileCommand(t, "delete", "go-backend"); strings.TrimSpace(output) != "Deleted profile go-backend" {
		t.Fatalf("delete output: %q", output)
	}
	if _, ok, err := environment.New(filepath.Join(root, "state.toml")).Load(); err != nil || ok {
		t.Fatalf("deleted profile target remains: target=%v error=%v", ok, err)
	}
}

func TestProfileUseRejectsMissingReferences(t *testing.T) {
	root := t.TempDir()
	setProfilePaths(
		t,
		filepath.Join(root, "active"),
		filepath.Join(root, "disabled"),
		filepath.Join(root, "groups"),
		filepath.Join(root, "profiles"),
	)

	storeDir := filepath.Join(root, "profiles")
	writeProfileForTest(t, storeDir, "broken", []string{"missing-group"}, []string{"missing-skill"}, nil)

	cmd := NewProfile()
	cmd.SetArgs([]string{"use", "broken"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing groups") {
		t.Fatalf("expected missing reference error, got %v", err)
	}
}

func executeProfileCommand(t *testing.T, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	cmd := NewProfile()
	cmd.SetArgs(args)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute profile %v: %v\n%s", args, err, output.String())
	}
	return output.String()
}

func setProfilePaths(t *testing.T, active, disabled, groups, profiles string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_GROUPS_DIR", groups)
	t.Setenv("SKILLER_PROFILES_DIR", profiles)
	t.Setenv("SKILLER_PINS_FILE", filepath.Join(filepath.Dir(disabled), "pins.toml"))
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(filepath.Dir(disabled), "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(filepath.Dir(disabled), "lock"))
}

func writeProfileForTest(t *testing.T, dir, name string, groups, skills, exclude []string) {
	t.Helper()
	store := profile.New(dir)
	if _, err := store.Create(name); err != nil {
		t.Fatalf("create test profile: %v", err)
	}
	if err := store.Update(name, profile.Profile{Name: name, Groups: groups, Skills: skills, Exclude: exclude}); err != nil {
		t.Fatalf("update test profile: %v", err)
	}
}

func assertProfilePathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
}
