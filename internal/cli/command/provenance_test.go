package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvenanceCommandShowsSkillLocation(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "active-skill")
	createSkill(t, disabledDir, "disabled-skill")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer
	cmd := NewProvenance()
	cmd.SetArgs([]string{"active-skill", "disabled-skill"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute provenance: %v", err)
	}

	text := output.String()
	for _, want := range []string{
		"SKILL",
		"LOCATION",
		"SOURCE",
		"active-skill",
		"active",
		"directory",
		"disabled-skill",
		"disabled",
		filepath.Join(activeDir, "active-skill"),
		filepath.Join(disabledDir, "disabled-skill"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("provenance output missing %q:\n%s", want, text)
		}
	}
}

func TestProvenanceCommandShowsBothConflictCopies(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "research")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer
	cmd := NewProvenance()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute conflict provenance: %v", err)
	}

	text := output.String()
	if got := strings.Count(text, "conflict"); got != 2 {
		t.Fatalf("expected two conflict rows, got %d occurrences:\n%s", got, text)
	}
	for _, want := range []string{
		"conflict",
		"active",
		"disabled",
		filepath.Join(activeDir, "research"),
		filepath.Join(disabledDir, "research"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("conflict provenance output missing %q:\n%s", want, text)
		}
	}
}
