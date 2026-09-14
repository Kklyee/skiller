package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	bubbletea "github.com/charmbracelet/bubbletea"
)

func TestMainViewAndKeyboardInteractions(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "beta"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})
	view := model.View()
	for _, want := range []string{"Skiller", "alpha", "beta", "coding", "Details", "active", "disabled"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'/'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("beta")})
	if got := len(model.visibleSkills()); got != 1 || model.visibleSkills()[0].ID != "beta" {
		t.Fatalf("search results: %+v", model.visibleSkills())
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{' '}})
	if skills, err := scanForTest(activeDir, disabledDir); err != nil {
		t.Fatalf("scan toggled skills: %v", err)
	} else if skills[0].State != catalog.StateActive || skills[1].State != catalog.StateActive {
		t.Fatalf("beta was not toggled: %+v", skills)
	}

	model, err = NewModel(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("reload model: %v", err)
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'d'}})
	if !strings.Contains(model.View(), "Skiller / Doctor") {
		t.Fatalf("doctor screen missing:\n%s", model.View())
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	if !strings.Contains(model.View(), "Skiller / Groups") {
		t.Fatalf("groups screen missing:\n%s", model.View())
	}
}

func TestConflictFromExternalInstallerIsVisible(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "research", "Research", "Active copy")
	createSkill(t, disabledDir, "research", "Research", "Disabled copy")

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Groups:   filepath.Join(root, "groups"),
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	view := model.View()
	for _, want := range []string{"Conflict 1", "research", "conflict", "!"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestReconcileModalAppliesGroup(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "old", "old", "")
	createSkill(t, disabledDir, "wanted", "wanted", "")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "wanted"); err != nil {
		t.Fatalf("add group skill: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyDown})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'u'}})
	if model.modal != modalReconcile {
		t.Fatal("expected reconcile modal")
	}
	if !strings.Contains(model.View(), "Enable") || !strings.Contains(model.View(), "Disable") {
		t.Fatalf("reconcile plan missing:\n%s", model.View())
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if model.modal != modalNone {
		t.Fatal("reconcile modal did not close")
	}
	if _, err := os.Stat(filepath.Join(activeDir, "wanted", "SKILL.md")); err != nil {
		t.Fatalf("wanted was not enabled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(disabledDir, "old", "SKILL.md")); err != nil {
		t.Fatalf("old was not disabled: %v", err)
	}
}

func TestResponsiveDetailsAndMinimumSize(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "alpha", "Alpha", "Description")
	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: filepath.Join(root, "disabled"),
		Groups:   filepath.Join(root, "groups"),
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 70, Height: 20})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if !strings.Contains(model.View(), "Description") {
		t.Fatalf("expanded details missing:\n%s", model.View())
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 40, Height: 20})
	if !strings.Contains(model.View(), "Terminal too small") {
		t.Fatalf("minimum size message missing:\n%s", model.View())
	}
}

func createSkill(t *testing.T, parent, id, name, description string) {
	t.Helper()
	dir := filepath.Join(parent, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	data := []byte("---\nname: " + name + "\ndescription: " + description + "\n---\n# Content\n")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), data, 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func scanForTest(active, disabled string) ([]catalog.Skill, error) {
	return catalog.Scan(active, disabled)
}
