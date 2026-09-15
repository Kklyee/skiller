package command

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/pin"
)

func TestPinListAndUnpinCommands(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	setPinPaths(t, activeDir, disabledDir, filepath.Join(root, "pins.toml"))

	var output bytes.Buffer
	cmd := NewPin()
	cmd.SetArgs([]string{"prototype"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pin skills: %v", err)
	}
	if !strings.Contains(output.String(), "Pinned 1 skills") {
		t.Fatalf("pin output: %q", output.String())
	}

	output.Reset()
	cmd = NewPins()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if !strings.Contains(output.String(), "prototype") || !strings.Contains(output.String(), "disabled") {
		t.Fatalf("pins output: %q", output.String())
	}

	output.Reset()
	cmd = NewUnpin()
	cmd.SetArgs([]string{"prototype"})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unpin skill: %v", err)
	}
	if !strings.Contains(output.String(), "Unpinned 1 skills") {
		t.Fatalf("unpin output: %q", output.String())
	}
}

func TestPinRejectsUninstalledSkill(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	setPinPaths(t, activeDir, disabledDir, filepath.Join(root, "pins.toml"))

	cmd := NewPin()
	cmd.SetArgs([]string{"missing-skill"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `skill "missing-skill" is not installed`) {
		t.Fatalf("pin unknown skill error = %v", err)
	}

	pinned, err := pin.New(filepath.Join(root, "pins.toml")).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 0 {
		t.Fatalf("pins after rejected command = %v", pinned)
	}
}

func TestPinSelectsInstalledSkillsInteractively(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)

	var output bytes.Buffer
	cmd := NewPin()
	cmd.SetIn(strings.NewReader("1,2\n"))
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive pin: %v", err)
	}
	if !strings.Contains(output.String(), "1) prototype") || !strings.Contains(output.String(), "2) research") || !strings.Contains(output.String(), "e.g. 1,3,5") || !strings.Contains(output.String(), "Pinned 2 skills") {
		t.Fatalf("interactive pin output: %q", output.String())
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 2 || pinned[0] != "prototype" || pinned[1] != "research" {
		t.Fatalf("interactive pins = %v", pinned)
	}
}

func TestPinSelectionOmitsAlreadyPinnedSkills(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)
	if _, err := pin.New(pinsPath).Add("prototype"); err != nil {
		t.Fatalf("create existing pin: %v", err)
	}

	var output bytes.Buffer
	cmd := NewPin()
	cmd.SetIn(strings.NewReader("1\n"))
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pin available skill: %v", err)
	}
	if !strings.Contains(output.String(), "1) research") || strings.Contains(output.String(), "prototype [disabled]") {
		t.Fatalf("available skill output: %q", output.String())
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 2 || pinned[0] != "prototype" || pinned[1] != "research" {
		t.Fatalf("pins after selecting available skill = %v", pinned)
	}
}

func TestUnpinSelectsCurrentPinsInteractively(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)
	if _, err := pin.New(pinsPath).Add("prototype", "research", "missing-skill"); err != nil {
		t.Fatalf("create pins: %v", err)
	}

	var output bytes.Buffer
	cmd := NewUnpin()
	cmd.SetIn(strings.NewReader("1,3\n"))
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive unpin: %v", err)
	}
	for _, want := range []string{"Pinned skills:", "1) missing-skill [missing]", "2) prototype [disabled]", "3) research [active]", "Unpinned 2 skills"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("interactive unpin output missing %q:\n%s", want, output.String())
		}
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 1 || pinned[0] != "prototype" {
		t.Fatalf("pins after interactive unpin = %v", pinned)
	}
}

func TestUnpinSelectionCanCancel(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)
	if _, err := pin.New(pinsPath).Add("research"); err != nil {
		t.Fatalf("create pin: %v", err)
	}

	var output bytes.Buffer
	cmd := NewUnpin()
	cmd.SetIn(strings.NewReader("q\n"))
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cancel unpin selection: %v", err)
	}
	if !strings.Contains(output.String(), "Cancelled") {
		t.Fatalf("cancel output: %q", output.String())
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 1 || pinned[0] != "research" {
		t.Fatalf("pins after cancellation = %v", pinned)
	}
}

func TestPinSelectionSupportsAll(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	createSkill(t, disabledDir, "prototype")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)

	cmd := NewPin()
	cmd.SetIn(strings.NewReader("a\n"))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("select all pins: %v", err)
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 2 || pinned[0] != "prototype" || pinned[1] != "research" {
		t.Fatalf("select all pins = %v", pinned)
	}
}

func TestPinSelectionRejectsInvalidNumber(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	setPinPaths(t, activeDir, disabledDir, filepath.Join(root, "pins.toml"))

	cmd := NewPin()
	cmd.SetIn(strings.NewReader("2\n"))
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "choose numbers from 1 to 1") {
		t.Fatalf("invalid selection error = %v", err)
	}
}

func TestPinRejectsMixedInstalledAndUninstalledSkills(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)

	cmd := NewPin()
	cmd.SetArgs([]string{"research", "missing-skill"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), `skill "missing-skill" is not installed`) {
		t.Fatalf("mixed pin error = %v", err)
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 0 {
		t.Fatalf("pins after mixed command = %v", pinned)
	}
}

func TestPinSelectionCanCancel(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research")
	pinsPath := filepath.Join(root, "pins.toml")
	setPinPaths(t, activeDir, disabledDir, pinsPath)

	var output bytes.Buffer
	cmd := NewPin()
	cmd.SetIn(strings.NewReader("q\n"))
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cancel pin selection: %v", err)
	}
	if !strings.Contains(output.String(), "Cancelled") {
		t.Fatalf("cancel output: %q", output.String())
	}

	pinned, err := pin.New(pinsPath).List()
	if err != nil {
		t.Fatalf("list pins: %v", err)
	}
	if len(pinned) != 0 {
		t.Fatalf("pins after cancellation = %v", pinned)
	}
}

func setPinPaths(t *testing.T, active, disabled, pins string) {
	t.Helper()
	t.Setenv("SKILLER_ACTIVE_DIR", active)
	t.Setenv("SKILLER_DISABLED_DIR", disabled)
	t.Setenv("SKILLER_PINS_FILE", pins)
}
