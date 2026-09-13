package cli

import (
	"bytes"
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

	cmd := newRootCommand()
	cmd.SetArgs([]string{"list"})
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
