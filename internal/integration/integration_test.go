package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/cli"
)

func TestProjectConfigurationWorkflow(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project", "service")
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	profilesDir := filepath.Join(root, "profiles")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	createSkill(t, activeDir, "old")
	createSkill(t, disabledDir, "wanted")
	if err := os.WriteFile(filepath.Join(projectDir, ".skiller.toml"), []byte("profile = \"backend\"\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)
	t.Setenv("SKILLER_GROUPS_DIR", groupsDir)
	t.Setenv("SKILLER_PROFILES_DIR", profilesDir)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(root, "transaction.json"))
	t.Setenv("SKILLER_LOCK", filepath.Join(root, "lock"))
	t.Chdir(projectDir)

	run(t, nil, "group", "create", "coding")
	run(t, nil, "group", "add", "coding", "wanted")
	run(t, nil, "profile", "create", "backend")
	run(t, nil, "profile", "edit", "backend", "--groups", "coding")

	syncOutput := run(t, strings.NewReader("y\n"), "sync")
	if !strings.Contains(syncOutput, "  + wanted") || !strings.Contains(syncOutput, "  - old") || !strings.Contains(syncOutput, "Synced ") {
		t.Fatalf("sync output:\n%s", syncOutput)
	}
	assertExists(t, filepath.Join(activeDir, "wanted", "SKILL.md"))
	assertExists(t, filepath.Join(disabledDir, "old", "SKILL.md"))

	statusOutput := run(t, nil, "status")
	assertStatus(t, statusOutput, "Active", "1")
	assertStatus(t, statusOutput, "Disabled", "1")
	assertStatus(t, statusOutput, "Conflict", "0")

	doctorOutput := run(t, nil, "doctor")
	if !strings.Contains(doctorOutput, "Status: healthy") {
		t.Fatalf("doctor output:\n%s", doctorOutput)
	}

	completionOutput := run(t, nil, "completion", "bash")
	if !strings.Contains(completionOutput, "__skiller_debug") {
		t.Fatalf("completion output missing bash function")
	}

	versionOutput := run(t, nil, "--version")
	if !strings.Contains(versionOutput, "skiller version dev") {
		t.Fatalf("version output: %q", versionOutput)
	}
}

func run(t *testing.T, input *strings.Reader, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	command := cli.NewRootCommand()
	command.SetArgs(args)
	if input != nil {
		command.SetIn(input)
	}
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, output.String())
	}
	return output.String()
}

func createSkill(t *testing.T, parent, id string) {
	t.Helper()
	dir := filepath.Join(parent, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Test Skill\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
}

func assertStatus(t *testing.T, output, label, value string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == label && fields[1] == value {
			return
		}
	}
	t.Fatalf("status %s %s not found:\n%s", label, value, output)
}
