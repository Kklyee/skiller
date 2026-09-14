package tui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/pin"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/charmbracelet/x/ansi"
)

func keyText(text string) bubbletea.KeyPressMsg {
	runes := []rune(text)
	var code rune
	if len(runes) > 0 {
		code = runes[0]
	}
	return bubbletea.KeyPressMsg{Text: text, Code: code}
}

func keyCode(code rune) bubbletea.KeyPressMsg {
	return bubbletea.KeyPressMsg{Code: code}
}

func viewText(model *Model) string {
	return ansi.Strip(model.View().Content)
}

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
	view := viewText(&model)
	for _, want := range []string{"Skill Environment Controller", "alpha", "beta", "coding", "Details", "Active", "Disabled"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}

	model.Update(keyText("/"))
	model.Update(keyText("beta"))
	if got := len(model.visibleSkills()); got != 1 || model.visibleSkills()[0].ID != "beta" {
		t.Fatalf("search results: %+v", model.visibleSkills())
	}
	model.Update(keyCode(bubbletea.KeyEsc))

	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyText(" "))
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
	model.Update(keyText("d"))
	if !strings.Contains(viewText(&model), "Skiller / Doctor") {
		t.Fatalf("doctor screen missing:\n%s", viewText(&model))
	}
	model.Update(keyCode(bubbletea.KeyEsc))
	model.Update(keyText("g"))
	if !strings.Contains(viewText(&model), "Skiller / Groups") {
		t.Fatalf("groups screen missing:\n%s", viewText(&model))
	}
}

func TestDetailsPanelShowsSkillProvenance(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	provenancePath := filepath.Join(root, "provenance.toml")
	createSkill(t, activeDir, "research", "Research", "Research skill")
	if err := skillprovenance.New(provenancePath).Set("research", skillprovenance.Entry{
		Source:     "github",
		Repository: "owner/repo",
		Installer:  "skills",
		Revision:   "abc123",
	}); err != nil {
		t.Fatalf("set provenance: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:     activeDir,
		Disabled:   filepath.Join(root, "disabled"),
		Provenance: provenancePath,
		Groups:     filepath.Join(root, "groups"),
		Journal:    filepath.Join(root, "transaction.json"),
		Lock:       filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	text := viewText(&model)
	for _, want := range []string{"Source: github", "Repository: owner/repo", "Installer: skills", "Revision: abc123"} {
		if !strings.Contains(text, want) {
			t.Fatalf("details missing %q:\n%s", want, text)
		}
	}
}

func TestInitStartsTerminalResizePolling(t *testing.T) {
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
	if model.Init() == nil {
		t.Fatal("Init returned no terminal resize polling command")
	}
}

func TestResizePollRoutesThroughWindowSizeMessage(t *testing.T) {
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

	model.Update(bubbletea.WindowSizeMsg{Width: 80, Height: 20})
	_, cmd := model.Update(resizePollMsg{width: 160, height: 40, valid: true})
	if model.width != 80 || model.height != 20 {
		t.Fatalf("resize poll bypassed WindowSizeMsg and changed model to %dx%d", model.width, model.height)
	}
	if cmd == nil {
		t.Fatal("resize poll returned no commands")
	}
	commandMessage := cmd()
	batch, ok := commandMessage.(bubbletea.BatchMsg)
	if !ok || len(batch) < 1 {
		t.Fatalf("resize poll command = %T, want BatchMsg", commandMessage)
	}
	firstMessage := batch[0]()
	message, ok := firstMessage.(bubbletea.WindowSizeMsg)
	if !ok {
		t.Fatalf("first resize command = %T, want WindowSizeMsg", firstMessage)
	}
	if message.Width != 160 || message.Height != 40 {
		t.Fatalf("window size message = %dx%d, want 160x40", message.Width, message.Height)
	}

	model.Update(message)
	if model.width != 160 || model.height != 40 {
		t.Fatalf("model size after WindowSizeMsg = %dx%d, want 160x40", model.width, model.height)
	}
}

func TestStartupLogoAnimationProgressesAndCanBeSkipped(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
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

	model.queueStartupAnimation()
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})
	if !model.startupActive {
		t.Fatal("startup animation did not start")
	}
	if view := viewText(&model); !strings.Contains(view, "·") {
		t.Fatalf("initial startup frame missing:\n%s", view)
	}

	model.Update(startupTickMsg{})
	if view := viewText(&model); !strings.Contains(view, "╭─╮") {
		t.Fatalf("logo frame missing after tick:\n%s", view)
	}
	model.Update(startupTickMsg{})
	if view := viewText(&model); !strings.Contains(view, "SKILLER") {
		t.Fatalf("brand missing after tick:\n%s", view)
	}

	model.Update(keyText("x"))
	if model.startupActive {
		t.Fatal("startup animation did not skip on key press")
	}
	if view := viewText(&model); !strings.Contains(view, "Installed") {
		t.Fatalf("main view did not resume after skip:\n%s", view)
	}
}

func TestStartupLogoAnimationSkipsWhenUnsupported(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
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

	model.queueStartupAnimation()
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})
	if model.startupActive || model.startupPending {
		t.Fatal("startup animation started with NO_COLOR")
	}

	t.Setenv("NO_COLOR", "")
	model.queueStartupAnimation()
	model.Update(bubbletea.WindowSizeMsg{Width: 50, Height: 30})
	if model.startupActive {
		t.Fatal("startup animation started in a small terminal")
	}
}

func TestMainColumnWidthsConstrainLargePanels(t *testing.T) {
	groupWidth, skillsWidth, detailsWidth := mainColumnWidths(180)
	if groupWidth != 30 || skillsWidth != 60 || detailsWidth != 90 {
		t.Fatalf("large layout widths = %d, %d, %d; want 30, 60, 90", groupWidth, skillsWidth, detailsWidth)
	}

	groupWidth, skillsWidth, detailsWidth = mainColumnWidths(90)
	if groupWidth != 22 || skillsWidth != 30 || detailsWidth != 38 {
		t.Fatalf("medium layout widths = %d, %d, %d; want 22, 30, 38", groupWidth, skillsWidth, detailsWidth)
	}
}

func TestMainUsesPersistentStartupLogoAboveSummary(t *testing.T) {
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

	lines := strings.Split(viewText(&model), "\n")
	logoStart := -1
	logoEnd := -1
	headerIndex := -1
	for index, line := range lines {
		if strings.Contains(line, "╭─╮") {
			logoStart = index
		}
		if strings.Contains(line, "Skill Environment Controller") {
			logoEnd = index
		}
		if strings.Contains(line, "Installed") {
			headerIndex = index
		}
	}
	if logoStart < 0 || logoEnd != logoStart+2 || headerIndex != logoEnd+1 {
		t.Fatalf("persistent logo/header rows are not stacked: start=%d end=%d header=%d\n%s", logoStart, logoEnd, headerIndex, viewText(&model))
	}
}

func TestMainPanelsReserveTheSameContentColumnAcrossFocus(t *testing.T) {
	column := mainColumn{content: "content", width: 30}
	focused := ansi.Strip(mainColumnBody(mainColumn{content: column.content, width: column.width, focused: true}, 8))
	unfocused := ansi.Strip(mainColumnBody(mainColumn{content: column.content, width: column.width, focused: false}, 8))
	contentColumn := func(view string) int {
		for _, line := range strings.Split(view, "\n") {
			if index := strings.Index(line, "content"); index >= 0 {
				return index
			}
		}
		return -1
	}
	if got, want := contentColumn(focused), contentColumn(unfocused); got != want {
		t.Fatalf("panel content moved between focus states: focused=%d unfocused=%d", got, want)
	}
}

func TestMainColumnBodiesClipOverflowToFixedHeight(t *testing.T) {
	content := strings.Join([]string{
		"one",
		"two",
		"three",
		"four",
		"five",
		"six",
		"seven",
		"eight",
		"nine",
		"ten",
	}, "\n")
	view := ansi.Strip(mainColumnBody(mainColumn{content: content, width: 30}, 8))

	if got, want := lipgloss.Height(view), 8; got != want {
		t.Fatalf("overflowing panel height = %d, want %d:\n%s", got, want, view)
	}
}

func TestMainViewColumnsKeepEqualHeightAfterShrink(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	names := []string{
		"ask-matt", "code-review", "codebase-design", "diagnosing-bugs", "domain-modeling",
		"find-skills", "grill-me", "grill-with-docs", "grilling", "handoff", "implement",
		"improve-codebase-architecture", "prototype", "research", "resolving-merge-conflicts",
		"setup-matt-pocock-skills", "show-me", "tdd", "teach", "to-questionnaire", "to-spec",
		"to-tickets", "triage", "wait-what", "wayfinder", "wizard", "writing-for-agents",
	}
	for _, name := range names {
		createSkill(t, activeDir, name, name, "Description")
	}
	model, err := NewModel(paths.Set{
		Active: activeDir, Disabled: filepath.Join(root, "disabled"), Groups: filepath.Join(root, "groups"),
		Journal: filepath.Join(root, "transaction.json"), Lock: filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 180, Height: 50})
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 30})
	view := viewText(&model)
	for _, line := range strings.Split(view, "\n") {
		if strings.Count(line, "╰")+strings.Count(line, "└") == 3 {
			return
		}
	}
	t.Fatalf("main column bottom borders do not share one row:\n%s", view)
}

func TestSkillsPanelScrollsSelectedSkillIntoView(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	for index := 0; index < 12; index++ {
		id := fmt.Sprintf("skill-%02d", index)
		createSkill(t, activeDir, id, id, "")
	}

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
	model.Update(bubbletea.WindowSizeMsg{Width: 120, Height: 18})
	model.Update(keyCode(bubbletea.KeyTab))

	initial := ansi.Strip(model.viewSkillsPanel())
	if !strings.Contains(initial, "skill-00") {
		t.Fatalf("initial skill window does not start at first skill:\n%s", initial)
	}

	for index := 0; index < 11; index++ {
		model.Update(keyCode(bubbletea.KeyDown))
	}
	scrolled := ansi.Strip(model.viewSkillsPanel())
	if !strings.Contains(scrolled, "skill-11") {
		t.Fatalf("scrolled skill window does not show selected skill:\n%s", scrolled)
	}
	if strings.Contains(scrolled, "skill-00") {
		t.Fatalf("scrolled skill window still shows first skill:\n%s", scrolled)
	}
}

func TestSkillRowsUseIconsWithoutRepeatedStateLabels(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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

	view := ansi.Strip(model.viewSkillsPanel())
	if !strings.Contains(view, "● Alpha [alpha]") || !strings.Contains(view, "○ Beta [beta]") {
		t.Fatalf("skill rows missing state icons:\n%s", view)
	}
	if strings.Contains(view, "active") || strings.Contains(view, "disabled") {
		t.Fatalf("skill rows repeat state labels:\n%s", view)
	}
}

func TestSelectedRowUsesBrandHighlight(t *testing.T) {
	if got, want := selectedRowStyle().GetForeground(), lipgloss.Color("6"); got != want {
		t.Fatalf("selected row color = %v, want %v", got, want)
	}
	if !selectedRowStyle().GetBold() {
		t.Fatal("selected row is not bold")
	}
}

func TestSearchBarShowsQueryAndMatchCount(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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
	model.Update(keyText("/"))
	if view := ansi.Strip(model.viewSkillsPanel()); !strings.Contains(view, "type to filter") || !strings.Contains(view, "2 matches") {
		t.Fatalf("empty search bar is unclear:\n%s", view)
	}

	model.Update(keyText("beta"))
	view := ansi.Strip(model.viewSkillsPanel())
	if !strings.Contains(view, "/ beta") || !strings.Contains(view, "1 match") {
		t.Fatalf("active search bar is missing query or count:\n%s", view)
	}
}

func TestDoctorViewUsesSemanticHealthSections(t *testing.T) {
	if got, want := doctorLevelStyle(doctor.Healthy).GetForeground(), lipgloss.Color("10"); got != want {
		t.Fatalf("healthy color = %v, want %v", got, want)
	}
	if got, want := doctorLevelStyle(doctor.Warning).GetForeground(), lipgloss.Color("11"); got != want {
		t.Fatalf("warning color = %v, want %v", got, want)
	}
	if got, want := doctorLevelStyle(doctor.Error).GetForeground(), lipgloss.Color("9"); got != want {
		t.Fatalf("error color = %v, want %v", got, want)
	}

	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatalf("create active directory: %v", err)
	}
	if err := os.MkdirAll(disabledDir, 0o755); err != nil {
		t.Fatalf("create disabled directory: %v", err)
	}
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
	model.Update(keyText("d"))
	view := viewText(&model)
	for _, want := range []string{"Paths", "Filesystem", "Skills", "Transactions", "Status:", "healthy"} {
		if !strings.Contains(view, want) {
			t.Fatalf("doctor view missing %q:\n%s", want, view)
		}
	}
}

func TestGroupDialogsUseDedicatedPanels(t *testing.T) {
	root := t.TempDir()
	groupsDir := filepath.Join(root, "groups")
	if _, err := group.New(groupsDir).Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	model, err := NewModel(paths.Set{
		Active:   filepath.Join(root, "active"),
		Disabled: filepath.Join(root, "disabled"),
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	model.Update(keyText("g"))
	model.Update(keyText("n"))
	view := viewText(&model)
	for _, want := range []string{"New Group", "Group name", "type name", "enter create"} {
		if !strings.Contains(view, want) {
			t.Fatalf("new group dialog missing %q:\n%s", want, view)
		}
	}

	model.Update(keyCode(bubbletea.KeyEsc))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("d"))
	view = viewText(&model)
	for _, want := range []string{"Confirm Delete", "Delete group coding?", "enter/y delete"} {
		if !strings.Contains(view, want) {
			t.Fatalf("delete dialog missing %q:\n%s", want, view)
		}
	}
}

func TestSpaceOnlyTogglesSkillsWhenSkillsFocused(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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

	model.Update(keyText(" "))
	skills, err := scanForTest(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan after groups-focused space: %v", err)
	}
	if skills[0].State != catalog.StateActive {
		t.Fatalf("groups-focused space changed alpha to %s", skills[0].State)
	}

	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyText(" "))
	skills, err = scanForTest(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan after skills-focused space: %v", err)
	}
	if skills[0].State != catalog.StateDisabled {
		t.Fatalf("skills-focused space did not toggle alpha: %s", skills[0].State)
	}
}

func TestSkillsFocusATogglesAllVisibleSkills(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")

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
	model.Update(keyCode(bubbletea.KeyTab))
	if footer := ansi.Strip(model.viewFooter()); !strings.Contains(footer, "a all") {
		t.Fatalf("skills footer missing select-all action:\n%s", footer)
	}

	model.Update(keyText("a"))
	if skills, err := scanForTest(activeDir, disabledDir); err != nil {
		t.Fatalf("scan after activate all: %v", err)
	} else {
		for _, skill := range skills {
			if skill.State != catalog.StateActive {
				t.Fatalf("after first a, %s state = %s, want active", skill.ID, skill.State)
			}
		}
	}

	model.Update(keyText("a"))
	if skills, err := scanForTest(activeDir, disabledDir); err != nil {
		t.Fatalf("scan after disable all: %v", err)
	} else {
		for _, skill := range skills {
			if skill.State != catalog.StateDisabled {
				t.Fatalf("after second a, %s state = %s, want disabled", skill.ID, skill.State)
			}
		}
	}
}

func TestAllVirtualGroupCanBeUsedAndMarked(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")
	store := group.New(groupsDir)
	if _, err := store.Create("hello"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("hello", "alpha"); err != nil {
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
	model.Update(keyText("u"))
	if model.modal != modalReconcile || model.plan.Group != "All" {
		t.Fatalf("using All did not open an All reconcile plan: modal=%v plan=%+v", model.modal, model.plan)
	}
	model.Update(keyCode(bubbletea.KeyEnter))

	skills, err := scanForTest(activeDir, disabledDir)
	if err != nil {
		t.Fatalf("scan after using All: %v", err)
	}
	for _, skill := range skills {
		if skill.State != catalog.StateActive {
			t.Fatalf("after using All, %s state = %s, want active", skill.ID, skill.State)
		}
	}
	view := ansi.Strip(model.viewGroupPanel())
	if !strings.Contains(view, "● All") {
		t.Fatalf("All group marker missing after using All:\n%s", view)
	}
	if strings.Contains(view, "Using All") {
		t.Fatalf("All status text should not be rendered:\n%s", view)
	}
}

func TestGroupPanelKeepsSelectionWithoutStatusLine(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "outside", "Outside", "Not in hello")
	createSkill(t, disabledDir, "inside", "Inside", "In hello")
	store := group.New(groupsDir)
	if _, err := store.Create("hello"); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := store.Add("hello", "inside"); err != nil {
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
	model.Update(keyCode(bubbletea.KeyDown))
	view := ansi.Strip(model.viewGroupPanel())
	if !strings.Contains(view, "›   hello") {
		t.Fatalf("selected hello group missing:\n%s", view)
	}
	if strings.Contains(view, "No group applied") || strings.Contains(view, "Using ") {
		t.Fatalf("group panel contains an extra status line:\n%s", view)
	}
}

func TestGroupNamesUseActiveGreen(t *testing.T) {
	if got, want := groupNameStyle().GetForeground(), lipgloss.Color("10"); got != want {
		t.Fatalf("group name color = %v, want %v", got, want)
	}
}

func TestGroupFocusEnterOpensGroupsPage(t *testing.T) {
	root := t.TempDir()
	groupsDir := filepath.Join(root, "groups")
	if _, err := group.New(groupsDir).Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   filepath.Join(root, "active"),
		Disabled: filepath.Join(root, "disabled"),
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	model.Update(keyCode(bubbletea.KeyDown))
	footer := ansi.Strip(model.viewFooter())
	if strings.Contains(footer, "space toggle") {
		t.Fatalf("groups footer advertises skill toggle:\n%s", footer)
	}
	if !strings.Contains(footer, "enter groups") {
		t.Fatalf("groups footer missing groups action:\n%s", footer)
	}
	model.Update(keyCode(bubbletea.KeyEnter))

	if model.screen != ScreenGroups {
		t.Fatalf("screen after groups-focused enter = %v, want groups", model.screen)
	}
	if model.detailExpanded {
		t.Fatal("groups-focused enter expanded skill details")
	}
	if !strings.Contains(viewText(&model), "Skiller / Groups") {
		t.Fatalf("groups page missing after enter:\n%s", viewText(&model))
	}
}

func TestGroupsPageShowsManagerColumnsAndUseAction(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, activeDir, "gamma", "Gamma", "Third skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")
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
	model.Update(keyText("g"))
	model.Update(keyCode(bubbletea.KeyDown))

	view := viewText(&model)
	for _, want := range []string{"Groups", "coding", "Members", "Alpha", "Activation Preview", "Keep 2", "Disable 1", "Not applied"} {
		if !strings.Contains(view, want) {
			t.Fatalf("groups manager missing %q:\n%s", want, view)
		}
	}

	model.Update(keyText("u"))
	if model.modal != modalReconcile {
		t.Fatalf("groups-page use did not open reconcile modal: %v", model.modal)
	}
}

func TestGroupsPageSelectsFirstGroupAndKeepsSelectionAtTop(t *testing.T) {
	root := t.TempDir()
	groupsDir := filepath.Join(root, "groups")
	store := group.New(groupsDir)
	for _, name := range []string{"hello", "test"} {
		if _, err := store.Create(name); err != nil {
			t.Fatalf("create group %s: %v", name, err)
		}
	}

	model, err := NewModel(paths.Set{
		Active:   filepath.Join(root, "active"),
		Disabled: filepath.Join(root, "disabled"),
		Groups:   groupsDir,
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	model.Update(keyText("g"))
	if model.screen != ScreenGroups {
		t.Fatalf("screen after opening groups = %v, want groups", model.screen)
	}
	if model.selectedGroup != "hello" {
		t.Fatalf("selected group after opening groups = %q, want hello", model.selectedGroup)
	}
	if strings.Contains(viewText(&model), "Select a group to inspect its members") {
		t.Fatalf("groups page opened without a selected group:\n%s", viewText(&model))
	}

	model.Update(keyCode(bubbletea.KeyUp))
	if model.selectedGroup != "hello" {
		t.Fatalf("selected group after moving above first group = %q, want hello", model.selectedGroup)
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
	view := viewText(&model)
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
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("u"))
	if model.modal != modalReconcile {
		t.Fatal("expected reconcile modal")
	}
	if !strings.Contains(viewText(&model), "Enable") || !strings.Contains(viewText(&model), "Disable") {
		t.Fatalf("reconcile plan missing:\n%s", viewText(&model))
	}
	if !strings.Contains(viewText(&model), "Summary") {
		t.Fatalf("reconcile plan summary missing:\n%s", viewText(&model))
	}
	model.Update(keyCode(bubbletea.KeyEnter))
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
	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyCode(bubbletea.KeyEnter))
	if !strings.Contains(viewText(&model), "Description") {
		t.Fatalf("expanded details missing:\n%s", viewText(&model))
	}
	model.Update(bubbletea.WindowSizeMsg{Width: 40, Height: 20})
	if !strings.Contains(viewText(&model), "Terminal too small") {
		t.Fatalf("minimum size message missing:\n%s", viewText(&model))
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

	if got, want := strings.Split(ansi.Strip(model.viewDetailsPanel()), "\n"), []string{
		"Name: Alpha",
		"Description: First skill",
		"Status: active",
		"Groups: coding",
		"Pinned: no",
	}; !slices.Equal(got, want) {
		t.Fatalf("details panel = %q, want %q", got, want)
	}

	model.Update(keyCode(bubbletea.KeyTab))
	if model.focus != FocusSkills {
		t.Fatalf("first tab focus = %v, want skills", model.focus)
	}
	model.Update(keyCode(bubbletea.KeyEnter))
	if model.focus != FocusSkills {
		t.Fatalf("enter focus = %v, want skills", model.focus)
	}
	model.Update(keyCode(bubbletea.KeyTab))
	if model.focus != FocusGroups {
		t.Fatalf("second tab focus = %v, want groups", model.focus)
	}
}

func TestTUIShowsPinnedSkillsAndKeepsThemActive(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	pinsPath := filepath.Join(root, "pins.toml")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")
	if _, err := pin.New(pinsPath).Add("beta"); err != nil {
		t.Fatalf("pin beta: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: disabledDir,
		Pins:     pinsPath,
		Groups:   filepath.Join(root, "groups"),
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyCode(bubbletea.KeyDown))

	view := viewText(&model)
	if !strings.Contains(view, "◆ ○ Beta [beta]") {
		t.Fatalf("pinned skill marker missing:\n%s", view)
	}
	if !strings.Contains(view, "Pinned: yes") {
		t.Fatalf("pinned detail missing:\n%s", view)
	}
	model.Update(keyText(" "))
	if _, err := os.Stat(filepath.Join(activeDir, "beta", "SKILL.md")); err != nil {
		t.Fatalf("pinned skill was not enabled: %v", err)
	}
	model.Update(keyText(" "))
	if _, err := os.Stat(filepath.Join(activeDir, "beta", "SKILL.md")); err != nil {
		t.Fatalf("pinned skill was disabled: %v", err)
	}
	if !strings.Contains(viewText(&model), "unpin it first") {
		t.Fatalf("pinned toggle message missing:\n%s", viewText(&model))
	}
}

func TestMultiSelectBatchVisibilityUsesTransaction(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, disabledDir, "beta", "Beta", "Second skill")
	createSkill(t, disabledDir, "gamma", "Gamma", "Third skill")

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
	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("x"))
	if !strings.Contains(viewText(&model), "1 selected") {
		t.Fatalf("selection count missing:\n%s", viewText(&model))
	}
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("x"))
	model.Update(keyText("b"))
	if model.modal != modalBatch {
		t.Fatalf("batch modal = %v, want batch modal", model.modal)
	}
	model.Update(keyText("e"))

	for _, id := range []string{"beta", "gamma"} {
		if _, err := os.Stat(filepath.Join(activeDir, id, "SKILL.md")); err != nil {
			t.Fatalf("batch did not enable %s: %v", id, err)
		}
	}
	if len(model.selectedSkills) != 0 {
		t.Fatalf("selection after batch = %v, want empty", model.selectedSkills)
	}
}

func TestMultiSelectBatchDisableKeepsPinnedSkillsActive(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	pinsPath := filepath.Join(root, "pins.toml")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, activeDir, "beta", "Beta", "Second skill")
	if _, err := pin.New(pinsPath).Add("beta"); err != nil {
		t.Fatalf("pin beta: %v", err)
	}

	model, err := NewModel(paths.Set{
		Active:   activeDir,
		Disabled: filepath.Join(root, "disabled"),
		Pins:     pinsPath,
		Groups:   filepath.Join(root, "groups"),
		Journal:  filepath.Join(root, "transaction.json"),
		Lock:     filepath.Join(root, "lock"),
	})
	if err != nil {
		t.Fatalf("new model: %v", err)
	}
	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyText("x"))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("x"))
	model.Update(keyText("b"))
	model.Update(keyText("d"))

	if _, err := os.Stat(filepath.Join(activeDir, "beta", "SKILL.md")); err != nil {
		t.Fatalf("pinned skill was disabled by batch action: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(activeDir), "disabled", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("unpinned skill was not disabled: %v", err)
	}
}

func TestMultiSelectBatchGroupAddAndRemove(t *testing.T) {
	root := t.TempDir()
	activeDir := filepath.Join(root, "active")
	disabledDir := filepath.Join(root, "disabled")
	groupsDir := filepath.Join(root, "groups")
	createSkill(t, activeDir, "alpha", "Alpha", "First skill")
	createSkill(t, activeDir, "beta", "Beta", "Second skill")
	store := group.New(groupsDir)
	if _, err := store.Create("coding"); err != nil {
		t.Fatalf("create group: %v", err)
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
	model.Update(keyCode(bubbletea.KeyTab))
	model.Update(keyText("x"))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("x"))
	model.Update(keyText("b"))
	model.Update(keyText("a"))
	if model.modal != modalBatchGroup {
		t.Fatalf("group batch modal = %v, want group batch modal", model.modal)
	}
	model.Update(keyCode(bubbletea.KeyEnter))

	created, err := store.Get("coding")
	if err != nil {
		t.Fatalf("read group after add: %v", err)
	}
	if want := []string{"alpha", "beta"}; !slices.Equal(created.Skills, want) {
		t.Fatalf("group skills after add = %v, want %v", created.Skills, want)
	}

	model.Update(keyCode(bubbletea.KeyUp))
	model.Update(keyText("x"))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText("x"))
	model.Update(keyText("b"))
	model.Update(keyText("r"))
	model.Update(keyCode(bubbletea.KeyEnter))
	created, err = store.Get("coding")
	if err != nil {
		t.Fatalf("read group after remove: %v", err)
	}
	if len(created.Skills) != 0 {
		t.Fatalf("group skills after remove = %v, want empty", created.Skills)
	}
}

func TestSkillStateBadgesUseSemanticColors(t *testing.T) {
	want := map[catalog.State]color.Color{
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

	model.Update(keyText("g"))
	model.Update(keyText("n"))
	model.Update(keyText("coding"))
	model.Update(keyCode(bubbletea.KeyEnter))

	if model.screen != ScreenGroupEditor {
		t.Fatalf("screen after group creation = %v, want group editor", model.screen)
	}
	for _, want := range []string{"Edit Group: coding", "alpha", "beta", "[ ]"} {
		if !strings.Contains(viewText(&model), want) {
			t.Fatalf("selector missing %q:\n%s", want, viewText(&model))
		}
	}

	model.Update(keyText(" "))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyText(" "))
	model.Update(keyCode(bubbletea.KeyEnter))

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
	model.Update(keyText("g"))
	model.Update(keyText("n"))
	model.Update(keyText("coding"))
	model.Update(keyCode(bubbletea.KeyEnter))
	model.Update(keyText("a"))
	model.Update(keyCode(bubbletea.KeyEnter))

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
	for _, line := range strings.Split(viewText(&model), "\n") {
		if strings.Contains(line, "Groups") && strings.Contains(line, "Skills") && strings.Contains(line, "Details") {
			header = line
			break
		}
	}
	if header == "" {
		t.Fatalf("single table header is missing:\n%s", viewText(&model))
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

	model.Update(keyText("g"))
	model.Update(keyText("n"))
	model.Update(keyCode(bubbletea.KeyEnter))

	if model.modal != modalGroupName {
		t.Fatalf("modal after empty group name = %v, want group name dialog", model.modal)
	}
	if !strings.Contains(viewText(&model), "Group name is required") {
		t.Fatalf("validation message missing:\n%s", viewText(&model))
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
	model.Update(keyText("g"))
	model.Update(keyCode(bubbletea.KeyDown))
	model.Update(keyCode(bubbletea.KeyEsc))
	model.Update(keyText("u"))
	model.Update(keyCode(bubbletea.KeyEnter))

	if model.modal != modalReconcile {
		t.Fatalf("modal after blocked reconcile = %v, want reconcile preview", model.modal)
	}
	if !strings.Contains(viewText(&model), "Cannot apply") {
		t.Fatalf("blocked reconcile message missing:\n%s", viewText(&model))
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

	model.Update(keyText("/"))
	if view := viewText(&model); !strings.Contains(view, "finish search") || !strings.Contains(view, "cancel") {
		t.Fatalf("search hints missing:\n%s", view)
	}

	model.Update(keyCode(bubbletea.KeyEsc))
	model.Update(keyText("g"))
	model.Update(keyText("n"))
	if view := viewText(&model); !strings.Contains(view, "type name") || !strings.Contains(view, "create") {
		t.Fatalf("group creation hints missing:\n%s", view)
	}
}

func TestMessageStylesUseSemanticColors(t *testing.T) {
	want := map[messageKind]color.Color{
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
	want := map[string]color.Color{
		"selected":   lipgloss.Color("10"),
		"unselected": lipgloss.Color("8"),
		"cursor":     lipgloss.Color("6"),
	}
	got := map[string]color.Color{
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

	if got := strings.Count(viewText(&model), "╭────────────────"); got != 1 {
		t.Fatalf("groups focus border count = %d, want 1:\n%s", got, viewText(&model))
	}
	model.Update(keyCode(bubbletea.KeyTab))
	if got := strings.Count(viewText(&model), "╭────────────────"); got != 1 {
		t.Fatalf("skills focus border count = %d, want 1:\n%s", got, viewText(&model))
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

	view := ansi.Strip(model.viewGroupPanel())
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
