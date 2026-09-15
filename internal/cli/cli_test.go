package cli

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRootCommandRejectsUnfinishedJournal(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "transaction.json")
	if err := os.WriteFile(journal, []byte("unfinished"), 0o600); err != nil {
		t.Fatalf("create journal: %v", err)
	}

	t.Setenv("SKILLER_ACTIVE_DIR", filepath.Join(root, "active"))
	t.Setenv("SKILLER_DISABLED_DIR", filepath.Join(root, "disabled"))
	t.Setenv("SKILLER_TRANSACTION_JOURNAL", journal)

	var output bytes.Buffer
	command := NewRootCommand()
	command.SetArgs([]string{"list"})
	command.SetOut(&output)
	command.SetErr(&output)

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "unfinished transaction journal") {
		t.Fatalf("unexpected journal error: %v", err)
	}
}

func TestRootVersionFlag(t *testing.T) {
	var output bytes.Buffer
	command := NewRootCommand()
	command.SetArgs([]string{"--version"})
	command.SetOut(&output)
	command.SetErr(&output)

	if err := command.Execute(); err != nil {
		t.Fatalf("execute version flag: %v", err)
	}
	if !strings.Contains(output.String(), "skiller version dev") {
		t.Fatalf("version output: %q", output.String())
	}
}

func TestRootCommandOmitsRemovedWorkflows(t *testing.T) {
	root := NewRootCommand()
	removed := map[string]bool{
		"export": false,
		"import": false,
		"sync":   false,
		"run":    false,
	}
	for _, command := range root.Commands() {
		if _, ok := removed[command.Name()]; ok {
			removed[command.Name()] = true
		}
	}
	for name, found := range removed {
		if found {
			t.Fatalf("removed command %q is still registered", name)
		}
	}
}

func TestRootCommandRegistersDelete(t *testing.T) {
	for _, command := range NewRootCommand().Commands() {
		if command.Name() == "delete" {
			return
		}
	}
	t.Fatal("delete command is not registered")
}

func TestHelpPaletteUsesSemanticColors(t *testing.T) {
	palette := newHelpPalette(&bytes.Buffer{})
	want := map[string]color.Color{
		"heading":  lipgloss.Color("6"),
		"command":  lipgloss.Color("10"),
		"argument": lipgloss.Color("11"),
		"flag":     lipgloss.Color("13"),
		"hint":     lipgloss.Color("8"),
	}

	got := map[string]color.Color{
		"heading":  palette.heading.GetForeground(),
		"command":  palette.command.GetForeground(),
		"argument": palette.argument.GetForeground(),
		"flag":     palette.flag.GetForeground(),
		"hint":     palette.hint.GetForeground(),
	}
	for name, wantColor := range want {
		if got[name] != wantColor {
			t.Fatalf("help %s color = %v, want %v", name, got[name], wantColor)
		}
	}
}
