package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestDetailsPanelIsReadOnlySummary(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")

	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "alpha"); err != nil {
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

	if got, want := strings.Split(model.viewDetailsPanel(), "\n"), []string{
		"Name: Alpha",
		"Description: First skill",
		"Status: active",
		"Groups: coding",
	}; !slices.Equal(got, want) {
		t.Fatalf("details panel = %q, want %q", got, want)
	}

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	if model.focus != FocusSkills {
		t.Fatalf("first tab focus = %v, want skills", model.focus)
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	if model.focus != FocusSkills {
		t.Fatalf("enter focus = %v, want skills", model.focus)
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	if model.focus != FocusGroups {
		t.Fatalf("second tab focus = %v, want groups", model.focus)
	}
}

func TestSkillStateBadgesUseSemanticColors(t *testing.T) {
	want := map[catalog.State]lipgloss.Color{
		catalog.StateActive:   lipgloss.Color("10"),
		catalog.StateDisabled: lipgloss.Color("8"),
		catalog.StateConflict: lipgloss.Color("9"),
		catalog.StateBroken:   lipgloss.Color("9"),
		catalog.StateInvalid:  lipgloss.Color("13"),
	}

	for state, color := range want {
		if got := stateStyle(state).GetForeground(); got != color {
			t.Fatalf("state %s color = %v, want %v", state, got, color)
		}
	}
}

func TestCreatingGroupOpensSkillSelector(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'n'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("coding")})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	if model.screen != ScreenGroupEditor {
		t.Fatalf("screen after group creation = %v, want group editor", model.screen)
	}
	for _, want := range []string{"Edit Group: coding", "alpha", "beta", "[ ]"} {
		if !strings.Contains(model.View(), want) {
			t.Fatalf("selector missing %q:\n%s", want, model.View())
		}
	}

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{' '}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyDown})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{' '}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	created, err := group.New(groupsDir).Get("coding")
	if err != nil {
		t.Fatalf("read created group: %v", err)
	}
	if want := []string{"alpha", "beta"}; !slices.Equal(created.Skills, want) {
		t.Fatalf("created group skills = %v, want %v", created.Skills, want)
	}
}

func TestGroupEditorSelectsAllSkillsWithA(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'n'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune("coding")})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'a'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	created, err := group.New(groupsDir).Get("coding")
	if err != nil {
		t.Fatalf("read created group: %v", err)
	}
	if want := []string{"alpha", "beta"}; !slices.Equal(created.Skills, want) {
		t.Fatalf("select all group skills = %v, want %v", created.Skills, want)
	}
}

func TestMainLayoutUsesSingleTableHeader(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")

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
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})

	var header string
	for _, line := range strings.Split(model.View(), "\n") {
		if strings.Contains(line, "Groups") && strings.Contains(line, "Skills") && strings.Contains(line, "Details") {
			header = line
			break
		}
	}
	if header == "" {
		t.Fatalf("single table header is missing:\n%s", model.View())
	}
	for _, border := range []string{"╭", "╮", "╰", "╯", "│"} {
		if strings.Contains(header, border) {
			t.Fatalf("table header contains panel border %q: %s", border, header)
		}
	}
}

func TestHelpViewUsesSemanticStyles(t *testing.T) {
	if got, want := helpKeyStyle().GetForeground(), lipgloss.Color("11"); got != want {
		t.Fatalf("help key color = %v, want %v", got, want)
	}
	if got, want := helpTextStyle().GetForeground(), lipgloss.Color("8"); got != want {
		t.Fatalf("help text color = %v, want %v", got, want)
	}
}

func TestEmptyGroupNameKeepsCreationDialogOpen(t *testing.T) {
	root := t.TempDir()
	model, err := NewModel(paths.Set{
		Active:   filepath.Join(root, "active"),
		Disabled: filepath.Join(root, "disabled"),
		Groups:   filepath.Join(root, "groups"),
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'n'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	if model.modal != modalGroupName {
		t.Fatalf("modal after empty group name = %v, want group name dialog", model.modal)
	}
	if !strings.Contains(model.View(), "Group name is required") {
		t.Fatalf("validation message missing:\n%s", model.View())
	}
}

func TestReconcileIssuesKeepPreviewOpen(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("coding", "missing"); err != nil {
		t.Fatalf("add missing skill: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: filepath.Join(root, "disabled"),
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyDown})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'u'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})

	if model.modal != modalReconcile {
		t.Fatalf("modal after blocked reconcile = %v, want reconcile preview", model.modal)
	}
	if !strings.Contains(model.View(), "Cannot apply") {
		t.Fatalf("blocked reconcile message missing:\n%s", model.View())
	}
}

func TestTUIShowsContextualKeyboardHints(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
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

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'/'}})
	if view := model.View(); !strings.Contains(view, "finish search") || !strings.Contains(view, "cancel") {
		t.Fatalf("search hints missing:\n%s", view)
	}

	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyEsc})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'g'}})
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'n'}})
	if view := model.View(); !strings.Contains(view, "type name") || !strings.Contains(view, "create") {
		t.Fatalf("group creation hints missing:\n%s", view)
	}
}

func TestMessageStylesUseSemanticColors(t *testing.T) {
	want := map[messageKind]lipgloss.Color{
		messageInfo:    lipgloss.Color("11"),
		messageSuccess: lipgloss.Color("10"),
		messageError:   lipgloss.Color("9"),
	}

	for kind, color := range want {
		if got := messageStyle(kind).GetForeground(); got != color {
			t.Fatalf("message %v color = %v, want %v", kind, got, color)
		}
	}
}

func TestEditorSelectionStylesUseSemanticColors(t *testing.T) {
	want := map[string]lipgloss.TerminalColor{
		"selected":   lipgloss.Color("10"),
		"unselected": lipgloss.Color("8"),
		"cursor":     lipgloss.Color("6"),
	}
	got := map[string]lipgloss.TerminalColor{
		"selected":   editorSelectedStyle().GetForeground(),
		"unselected": editorUnselectedStyle().GetForeground(),
		"cursor":     editorCursorStyle().GetForeground(),
	}
	for name, wantColor := range want {
		if got[name] != wantColor {
			t.Fatalf("editor %s color = %v, want %v", name, got[name], wantColor)
		}
	}
}

func TestFocusedPanelsHaveVisibleBorders(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
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
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})

	if got := strings.Count(model.View(), "╭"); got != 1 {
		t.Fatalf("groups focus border count = %d, want 1:\n%s", got, model.View())
	}
	model.Update(bubbletea.KeyMsg{Type: bubbletea.KeyTab})
	if got := strings.Count(model.View(), "╭"); got != 1 {
		t.Fatalf("skills focus border count = %d, want 1:\n%s", got, model.View())
	}
}

func TestGroupsShowCurrentlyActiveGroup(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

	store := group.New(groupsDir)
	for _, name := range []string{"coding", "other"} {
		if _, err := store.Create(name); err != nil {
			t.Fatalf("create group %s: %v", name, err)
		}
	}
	if _, err := store.Add("coding", "alpha"); err != nil {
		t.Fatalf("add coding skill: %v", err)
	}
	if _, err := store.Add("other", "beta"); err != nil {
		t.Fatalf("add other skill: %v", err)
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

	view := model.viewGroupPanel()
	if !strings.Contains(view, "● coding") {
		t.Fatalf("active group marker missing:\n%s", view)
	}
	if strings.Contains(view, "● other") {
		t.Fatalf("inactive group was marked active:\n%s", view)
	}
	header := model.viewHeader()
	for _, want := range []string{"Installed", "Active", "Disabled", "Conflict", "Group:"} {
		if !strings.Contains(header, want) {
			t.Fatalf("header missing %q:\n%s", want, header)
		}
	}
	for _, unwanted := range []string{"Broken", "Invalid", "Using:", "● coding"} {
		if strings.Contains(header, unwanted) {
			t.Fatalf("header contains unwanted %q:\n%s", unwanted, header)
		}
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
