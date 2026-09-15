package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	createSkill(t, activeDir, "research")
	createSkill(t, activeDir, "code-review")
	createSkill(t, disabledDir, "prototype")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer

	cmd := NewList()
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute list command: %v", err)
	}

	lines := strings.Split(
		strings.TrimSpace(output.String()),
		"\n",
	)

	if len(lines) != 4 {
		t.Fatalf(
			"expected 4 output lines, got %d:\n%s",
			len(lines),
			output.String(),
		)
	}

	assertFields(t, lines[0], "SKILL", "STATUS")
	assertFields(t, lines[1], "code-review", "active")
	assertFields(t, lines[2], "prototype", "disabled")
	assertFields(t, lines[3], "research", "active")
}

func TestListCommandSurfacesInstallerConflict(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer
	cmd := NewList()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute list command: %v", err)
	}
	if !strings.Contains(output.String(), "research") || !strings.Contains(output.String(), "conflict") {
		t.Fatalf("conflict missing from list:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("active copy was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "research", "SKILL.md")); err != nil {
		t.Fatalf("disabled copy was removed: %v", err)
	}
}

func TestListCommandJSON(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer
	cmd := NewList()
	cmd.SetArgs([]string{"--json"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute JSON list: %v", err)
	}
	var rows []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatalf("decode list JSON: %v\n%s", err, output.String())
	}
	if len(rows) != 2 || rows[0].ID != "prototype" || rows[0].Status != "disabled" || rows[1].ID != "research" || rows[1].Status != "active" {
		t.Fatalf("list JSON: %#v", rows)
	}
}

func createSkill(
	t *testing.T,
	parent string,
	name string,
) {
	t.Helper()

	dir := filepath.Join(parent, name)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}

	skillFile := filepath.Join(dir, "SKILL.md")

	if err := os.WriteFile(
		skillFile,
		[]byte("# Test Skill\n"),
		0o644,
	); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func assertFields(
	t *testing.T,
	line string,
	want ...string,
) {
	t.Helper()

	got := strings.Fields(line)

	if len(got) != len(want) {
		t.Fatalf(
			"expected fields %q, got %q",
			want,
			got,
		)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf(
				"field %d: expected %q, got %q",
				i,
				want[i],
				got[i],
			)
		}
	}
}
