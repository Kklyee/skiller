package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestGroupCommands(t *testing.T) {
	root := t.TempDir()
	groupsDir := filepath.Join(root, "groups")
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "code-review")
	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", filepath.Join(root, "disabled"))
	t.Setenv("SKILLER_GROUPS_DIR", groupsDir)

	output := executeGroupCommand(t, "create", "coding")
	if strings.TrimSpace(output) != "Created group coding" {
		t.Fatalf("create output: %q", output)
	}

	output = executeGroupCommand(t, "add", "coding", "code-review", "missing")
	if strings.TrimSpace(output) != "Added 2 skills to coding" {
		t.Fatalf("add output: %q", output)
	}

	output = executeGroupCommand(t, "list")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("list output: %q", output)
	}
	assertFields(t, lines[0], "GROUP", "SKILLS", "MISSING")
	assertFields(t, lines[1], "coding", "2", "missing")

	output = executeGroupCommand(t, "show", "coding")
	if !strings.Contains(output, "Name: coding") || !strings.Contains(output, "  missing") {
		t.Fatalf("show output: %q", output)
	}

	output = executeGroupCommand(t, "remove", "coding", "missing")
	if strings.TrimSpace(output) != "Removed 1 skills from coding" {
		t.Fatalf("remove output: %q", output)
	}

	output = executeGroupCommand(t, "delete", "coding")
	if strings.TrimSpace(output) != "Deleted group coding" {
		t.Fatalf("delete output: %q", output)
	}
}

func executeGroupCommand(t *testing.T, args ...string) string {
	t.Helper()

	var output bytes.Buffer
	command := NewGroup()
	command.SetArgs(args)
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute group %v: %v", args, err)
	}

	return output.String()
}
