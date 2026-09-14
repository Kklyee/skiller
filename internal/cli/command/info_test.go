package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
)

func TestInfoCommandShowsSkillProvenance(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	provenancePath := filepath.Join(root, "provenance.toml")
	createSkill(t, activeDir, "code-review")
	if err := skillprovenance.New(provenancePath).Set("code-review", skillprovenance.Entry{
		Source:     "github",
		Repository: "owner/repo",
		Installer:  "skills",
		Revision:   "abc123",
	}); err != nil {
		t.Fatalf("set provenance: %v", err)
	}

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", filepath.Join(root, "disabled"))
	t.Setenv("SKILLER_PROVENANCE_FILE", provenancePath)

	var output bytes.Buffer
	cmd := NewInfo()
	cmd.SetArgs([]string{"code-review"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute info: %v", err)
	}

	for _, want := range []string{
		"Skill: code-review",
		"Status: active",
		"Source: github",
		"Repository: owner/repo",
		"Installer: skills",
		"Revision: abc123",
		"Location: active",
		"Source type: directory",
		filepath.Join(activeDir, "code-review"),
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("info output missing %q:\n%s", want, output.String())
		}
	}
}

func TestInfoCommandShowsConflictLocations(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "research")
	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)

	var output bytes.Buffer
	cmd := NewInfo()
	cmd.SetArgs([]string{"research"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute conflict info: %v", err)
	}

	text := output.String()
	for _, want := range []string{
		"Status: conflict",
		"Locations:",
		"active",
		"disabled",
		filepath.Join(activeDir, "research"),
		filepath.Join(disabledDir, "research"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("conflict info missing %q:\n%s", want, text)
		}
	}
}

func TestInfoCommandRejectsMissingSkill(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SKILLER_ACTIVE_DIR", filepath.Join(root, "active"))
	t.Setenv("SKILLER_DISABLED_DIR", filepath.Join(root, "disabled"))
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}

	cmd := NewInfo()
	cmd.SetArgs([]string{"missing"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("missing skill error: %v", err)
	}
}
