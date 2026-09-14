package command

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSynchronizesBeforeLaunchingAgent(t *testing.T) {
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

	var launched string
	var output bytes.Buffer
	runAgent := func(name string, in io.Reader, out, errOut io.Writer) error {
		launched = name
		_, err := fmt.Fprintln(out, "agent started")
		return err
	}
	cmd := newRun(runAgent)
	cmd.SetArgs([]string{"codex"})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("run codex: %v\n%s", err, output.String())
	}
	if launched != "codex" {
		t.Fatalf("launched agent: %q", launched)
	}
	if !strings.Contains(output.String(), "Synced ") || !strings.Contains(output.String(), "Launching codex") || !strings.Contains(output.String(), "agent started") {
		t.Fatalf("run output: %q", output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not enabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old was not disabled: %v", err)
	}
}

func TestRunLaunchesWithoutChanges(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "wanted")
	writeProjectConfig(t, projectDir, "skills = [\"wanted\"]\n")
	setSyncPaths(t, activeDir, filepath.Join(root, "disabled"), filepath.Join(root, "groups"), filepath.Join(root, "profiles"))
	t.Chdir(projectDir)

	var launched string
	var output bytes.Buffer
	cmd := newRun(func(name string, in io.Reader, out, errOut io.Writer) error {
		launched = name
		return nil
	})
	cmd.SetArgs([]string{"gemini"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("run gemini: %v\n%s", err, output.String())
	}
	if launched != "gemini" || strings.Contains(output.String(), "Apply plan?") {
		t.Fatalf("run state: agent=%q output=%q", launched, output.String())
	}
}

func TestRunRestoreRestoresOriginalActiveState(t *testing.T) {
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
	runAgent := func(name string, in io.Reader, out, errOut io.Writer) error {
		if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
			t.Fatalf("project skill was not active during run: %v", err)
		}
		if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
			t.Fatalf("original skill was not disabled during run: %v", err)
		}
		return nil
	}
	cmd := newRun(runAgent)
	cmd.SetArgs([]string{"codex", "--restore"})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("run with restore: %v\n%s", err, output.String())
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("original skill was not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("project skill was not restored: %v", err)
	}
	if !strings.Contains(output.String(), "Restored original skill environment") {
		t.Fatalf("restore output missing: %q", output.String())
	}
}

func TestRunRestoreRestoresWhenAgentFails(t *testing.T) {
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

	cmd := newRun(func(name string, in io.Reader, out, errOut io.Writer) error {
		return errors.New("agent failed")
	})
	cmd.SetArgs([]string{"codex", "--restore"})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "agent failed") {
		t.Fatalf("agent error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("original skill was not restored after agent failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("project skill was not restored after agent failure: %v", err)
	}
}

func TestRunCancelsBeforeLaunching(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "old")
	writeProjectConfig(t, projectDir, "skills = []\n")
	setSyncPaths(t, activeDir, filepath.Join(root, "disabled"), filepath.Join(root, "groups"), filepath.Join(root, "profiles"))
	t.Chdir(projectDir)

	launched := false
	var output bytes.Buffer
	cmd := newRun(func(name string, in io.Reader, out, errOut io.Writer) error {
		launched = true
		return nil
	})
	cmd.SetArgs([]string{"opencode"})
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cancel run: %v\n%s", err, output.String())
	}
	if launched || !strings.Contains(output.String(), "Cancelled") {
		t.Fatalf("cancel state: launched=%t output=%q", launched, output.String())
	}
}

func TestRunRejectsUnknownAgent(t *testing.T) {
	called := false
	cmd := newRun(func(name string, in io.Reader, out, errOut io.Writer) error {
		called = true
		return nil
	})
	cmd.SetArgs([]string{"unknown"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported coding agent") {
		t.Fatalf("unknown agent error: %v", err)
	}
	if called {
		t.Fatal("unknown agent was launched")
	}
}
