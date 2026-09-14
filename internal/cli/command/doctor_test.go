package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommandHealthy(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "active")
	createSkill(t, disabledDir, "disabled")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(root, "transaction.json"))

	var output bytes.Buffer
	cmd := NewDoctor()
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute doctor: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "Status: healthy") {
		t.Fatalf("expected healthy status, got:\n%s", output.String())
	}
}

func TestDoctorCommandReturnsErrorForConflict(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "conflict")
	createSkill(t, disabledDir, "conflict")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(root, "transaction.json"))

	var output bytes.Buffer
	cmd := NewDoctor()
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected doctor error")
	}
	if !strings.Contains(output.String(), "Status: error") {
		t.Fatalf("expected error status, got:\n%s", output.String())
	}
}

func TestDoctorCommandDoesNotCreateMissingDirectories(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")

	t.Setenv("SKILLER_ACTIVE_DIR", activeDir)
	t.Setenv("SKILLER_DISABLED_DIR", disabledDir)
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", filepath.Join(root, "transaction.json"))

	cmd := NewDoctor()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
	if _, err := os.Stat(activeDir); !os.IsNotExist(err) {
		t.Fatalf("doctor created active directory: %v", err)
	}
	if _, err := os.Stat(disabledDir); !os.IsNotExist(err) {
		t.Fatalf("doctor created disabled directory: %v", err)
	}
}
