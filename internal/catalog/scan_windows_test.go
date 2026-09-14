package catalog

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScanDirectoryJunction(t *testing.T) {
	root := t.TempDir()

	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	targetDir := filepath.Join(root, "target")

	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}

	createTestSkill(t, targetDir, "source")

	junctionPath := filepath.Join(activeDir, "junctioned")
	if output, err := exec.Command(
		"cmd.exe",
		"/c",
		"mklink",
		"/J",
		junctionPath,
		filepath.Join(targetDir, "source"),
	).CombinedOutput(); err != nil {
		t.Skipf("directory junctions are unavailable: %v: %s", err, output)
	}
	skills, err := Scan(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan skills: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	assertSkillState(t, skills[0], "junctioned", StateActive)
	if skills[0].ActiveSource != SourceJunction {
		t.Fatalf("expected junction source, got %s", skills[0].ActiveSource)
	}
}
