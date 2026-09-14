package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/Kklyee/skiller/internal/visibility"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen uint8

const (
	ScreenMain Screen = iota + 1
	ScreenGroups
	ScreenGroupEditor
	ScreenGroupDetails
	ScreenDoctor
	ScreenHelp
)

type Focus uint8

const (
	FocusGroups Focus = iota + 1
	FocusSkills
	FocusDetails
)

type modal uint8

const (
	modalNone modal = iota
	modalReconcile
	modalDeleteGroup
	modalGroupName
)

type Model struct {
	paths paths.Set

	skills  []catalog.Skill
	groups  []group.Group
	summary catalog.Summary

	screen Screen
	focus  Focus

	selectedGroup  string
	selectedSkill  string
	detailExpanded bool

	search       string
	searchActive bool

	modal       modal
	plan        reconcile.Plan
	input       string
	deleteGroup string

	editorGroup  group.Group
	editorSkills []string
	editorChosen map[string]bool
	editorIndex  int

	doctorReport doctor.Report
	message      string
	width        int
	height       int
}

func NewModel(pathSet paths.Set) (Model, error) {
	model := Model{
		paths:  pathSet,
		screen: ScreenMain,
		focus:  FocusGroups,
		width:  120,
		height: 30,
	}
	if err := model.refresh(); err != nil {
		return Model{}, err
	}

	return model, nil
}

func Run(pathSet paths.Set) error {
	model, err := NewModel(pathSet)
	if err != nil {
		return err
	}

	_, err = bubbletea.NewProgram(&model, bubbletea.WithAltScreen()).Run()
	return err
}

func (m *Model) Init() bubbletea.Cmd {
	return nil
}

func (m *Model) Update(message bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch message := message.(type) {
	case bubbletea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
	case bubbletea.KeyMsg:
		return m, m.updateKey(message)
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width < 60 || m.height < 8 {
		return "Terminal too small. Resize to at least 60x8.\n"
	}
	if m.modal == modalReconcile {
		return m.viewReconcileModal()
	}

	switch m.screen {
	case ScreenGroups:
		return m.viewGroups()
	case ScreenGroupEditor:
		return m.viewGroupEditor()
	case ScreenGroupDetails:
		return m.viewGroupDetails()
	case ScreenDoctor:
		return m.viewDoctor()
	case ScreenHelp:
		return m.viewHelp()
	default:
		return m.viewMain()
	}
}

func (m *Model) refresh() error {
	previousGroup := m.selectedGroup
	previousSkill := m.selectedSkill

	skills, err := catalog.Scan(m.paths.Active, m.paths.Disabled)
	if err != nil {
		return err
	}
	groups, err := group.New(m.paths.Groups).ListWithMissing(skills)
	if err != nil {
		return err
	}

	m.skills = skills
	m.groups = groups
	m.summary = catalog.Summarize(skills)
	m.summary.ActiveDir = m.paths.Active
	m.summary.DisabledDir = m.paths.Disabled
	m.selectedGroup = previousGroup
	m.selectedSkill = previousSkill
	m.normalizeSelection()

	return nil
}

func (m *Model) updateKey(message bubbletea.KeyMsg) bubbletea.Cmd {
	if m.modal != modalNone {
		return m.updateModal(message)
	}
	if m.searchActive {
		return m.updateSearch(message)
	}
	if m.screen == ScreenGroupEditor {
		return m.updateEditor(message)
	}

	key := message.String()
	switch m.screen {
	case ScreenGroups:
		return m.updateGroups(message, key)
	case ScreenGroupDetails, ScreenDoctor, ScreenHelp:
		if key == "esc" || key == "q" {
			m.screen = ScreenMain
			m.message = ""
		}
		return nil
	}

	if key == "ctrl+c" || key == "q" {
		return bubbletea.Quit
	}

	switch key {
	case "tab":
		m.focus++
		if m.focus > FocusDetails {
			m.focus = FocusGroups
		}
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case " ":
		m.toggleSelectedSkill()
	case "/":
		m.searchActive = true
		m.search = ""
	case "enter":
		m.detailExpanded = true
		m.focus = FocusDetails
	case "esc":
		m.detailExpanded = false
		m.focus = FocusSkills
	case "g":
		m.screen = ScreenGroups
		m.message = ""
	case "u":
		m.openReconcile()
	case "d":
		m.doctorReport = doctor.Inspect(m.paths)
		m.screen = ScreenDoctor
	case "?":
		m.screen = ScreenHelp
		m.message = ""
	}

	return nil
}

func (m *Model) updateSearch(message bubbletea.KeyMsg) bubbletea.Cmd {
	switch message.Type {
	case bubbletea.KeyEsc:
		m.searchActive = false
		m.search = ""
	case bubbletea.KeyEnter:
		m.searchActive = false
	case bubbletea.KeyBackspace, bubbletea.KeyDelete:
		runes := []rune(m.search)
		if len(runes) > 0 {
			m.search = string(runes[:len(runes)-1])
		}
	case bubbletea.KeyRunes:
		m.search += string(message.Runes)
	}
	m.normalizeSelection()
	return nil
}

func (m *Model) updateGroups(message bubbletea.KeyMsg, key string) bubbletea.Cmd {
	switch key {
	case "ctrl+c", "q":
		return bubbletea.Quit
	case "esc":
		m.screen = ScreenMain
		m.message = ""
	case "up", "k":
		m.moveGroup(-1)
	case "down", "j":
		m.moveGroup(1)
	case "n":
		m.modal = modalGroupName
		m.input = ""
	case "e":
		m.openEditor()
	case "d":
		m.openDeleteGroup()
	case "enter":
		if m.selectedGroup != "" {
			m.screen = ScreenGroupDetails
		}
	}
	return nil
}

func (m *Model) updateEditor(message bubbletea.KeyMsg) bubbletea.Cmd {
	key := message.String()
	switch key {
	case "esc":
		m.screen = ScreenGroups
		m.message = ""
	case "up", "k":
		if m.editorIndex > 0 {
			m.editorIndex--
		}
	case "down", "j":
		if m.editorIndex+1 < len(m.editorSkills) {
			m.editorIndex++
		}
	case " ":
		if len(m.editorSkills) > 0 {
			id := m.editorSkills[m.editorIndex]
			m.editorChosen[id] = !m.editorChosen[id]
		}
	case "s":
		m.saveEditor()
	}
	return nil
}

func (m *Model) updateModal(message bubbletea.KeyMsg) bubbletea.Cmd {
	key := message.String()
	if m.modal == modalReconcile {
		if key == "esc" {
			m.modal = modalNone
			return nil
		}
		if key == "enter" {
			if m.plan.HasIssues() {
				m.message = "Cannot apply: resolve missing skills or catalog issues first"
				m.modal = modalNone
				return nil
			}
			if err := transaction.Apply(m.paths, m.plan); err != nil {
				m.message = err.Error()
			} else if err := m.refresh(); err != nil {
				m.message = err.Error()
			} else {
				m.message = fmt.Sprintf("Applied group %s", m.plan.Group)
			}
			m.modal = modalNone
		}
		return nil
	}

	if m.modal == modalGroupName {
		switch message.Type {
		case bubbletea.KeyEsc:
			m.modal = modalNone
		case bubbletea.KeyEnter:
			name := strings.TrimSpace(m.input)
			if name == "" {
				m.message = "Group name is required"
				m.modal = modalNone
				return nil
			}
			if _, err := group.New(m.paths.Groups).Create(name); err != nil {
				m.message = err.Error()
			} else if err := m.refresh(); err != nil {
				m.message = err.Error()
			} else {
				m.selectedGroup = name
				m.message = fmt.Sprintf("Created group %s", name)
			}
			m.modal = modalNone
		case bubbletea.KeyBackspace, bubbletea.KeyDelete:
			runes := []rune(m.input)
			if len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		case bubbletea.KeyRunes:
			m.input += string(message.Runes)
		}
		return nil
	}

	if key == "esc" || key == "n" {
		m.modal = modalNone
		return nil
	}
	if key == "enter" || strings.EqualFold(key, "y") {
		if err := group.New(m.paths.Groups).Delete(m.deleteGroup); err != nil {
			m.message = err.Error()
		} else if err := m.refresh(); err != nil {
			m.message = err.Error()
		} else {
			m.selectedGroup = ""
			m.message = fmt.Sprintf("Deleted group %s", m.deleteGroup)
		}
		m.modal = modalNone
	}
	return nil
}

func (m *Model) moveSelection(delta int) {
	if m.focus == FocusGroups {
		m.moveGroup(delta)
		return
	}
	if m.focus != FocusSkills {
		return
	}

	visible := m.visibleSkills()
	if len(visible) == 0 {
		m.selectedSkill = ""
		return
	}
	index := 0
	for i, skill := range visible {
		if skill.ID == m.selectedSkill {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(visible) {
		index = len(visible) - 1
	}
	m.selectedSkill = visible[index].ID
}

func (m *Model) moveGroup(delta int) {
	if len(m.groups) == 0 {
		m.selectedGroup = ""
		return
	}
	index := -1
	if m.selectedGroup != "" {
		for i, group := range m.groups {
			if group.Name == m.selectedGroup {
				index = i
				break
			}
		}
	}
	index += delta
	if index < -1 {
		index = -1
	}
	if index >= len(m.groups) {
		index = len(m.groups) - 1
	}
	if index == -1 {
		m.selectedGroup = ""
		return
	}
	m.selectedGroup = m.groups[index].Name
	m.normalizeSelection()
}

func (m *Model) normalizeSelection() {
	visible := m.visibleSkills()
	if len(visible) == 0 {
		m.selectedSkill = ""
		return
	}
	for _, skill := range visible {
		if skill.ID == m.selectedSkill {
			return
		}
	}
	m.selectedSkill = visible[0].ID
}

func (m *Model) visibleSkills() []catalog.Skill {
	groupIDs := map[string]bool(nil)
	if m.selectedGroup != "" {
		groupIDs = make(map[string]bool)
		for _, group := range m.groups {
			if group.Name == m.selectedGroup {
				for _, id := range group.Skills {
					groupIDs[id] = true
				}
				break
			}
		}
	}

	query := strings.ToLower(m.search)
	visible := make([]catalog.Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		if groupIDs != nil && !groupIDs[skill.ID] {
			continue
		}
		if query != "" && !m.matches(skill, query) {
			continue
		}
		visible = append(visible, skill)
	}

	return visible
}

func (m *Model) matches(skill catalog.Skill, query string) bool {
	values := []string{skill.ID, skill.Name, skill.Description}
	for _, group := range m.groups {
		for _, id := range group.Skills {
			if id == skill.ID {
				values = append(values, group.Name)
				break
			}
		}
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func (m *Model) selectedSkillValue() (catalog.Skill, bool) {
	for _, skill := range m.visibleSkills() {
		if skill.ID == m.selectedSkill {
			return skill, true
		}
	}
	return catalog.Skill{}, false
}

func (m *Model) toggleSelectedSkill() {
	skill, ok := m.selectedSkillValue()
	if !ok {
		return
	}

	var err error
	switch skill.State {
	case catalog.StateActive:
		_, err = visibility.Disable(m.paths.Active, m.paths.Disabled, skill.ID)
	case catalog.StateDisabled:
		_, err = visibility.Enable(m.paths.Active, m.paths.Disabled, skill.ID)
	default:
		err = fmt.Errorf("cannot toggle %s skill %q", skill.State, skill.ID)
	}
	if err != nil {
		m.message = err.Error()
		return
	}
	if err := m.refresh(); err != nil {
		m.message = err.Error()
		return
	}
	m.message = fmt.Sprintf("Toggled %s", skill.ID)
}

func (m *Model) openReconcile() {
	if m.selectedGroup == "" {
		m.message = "Select a group before pressing u"
		return
	}
	for _, group := range m.groups {
		if group.Name == m.selectedGroup {
			m.plan = reconcile.Build(group, m.skills)
			m.modal = modalReconcile
			return
		}
	}
}

func (m *Model) openEditor() {
	if m.selectedGroup == "" {
		m.message = "Select a group before editing"
		return
	}
	for _, group := range m.groups {
		if group.Name != m.selectedGroup {
			continue
		}
		m.editorGroup = group
		m.editorSkills = make([]string, 0, len(m.skills))
		for _, skill := range m.skills {
			m.editorSkills = append(m.editorSkills, skill.ID)
		}
		known := make(map[string]bool, len(m.editorSkills))
		for _, id := range m.editorSkills {
			known[id] = true
		}
		for _, id := range group.Skills {
			if !known[id] {
				m.editorSkills = append(m.editorSkills, id)
			}
		}
		slices.Sort(m.editorSkills)
		m.editorChosen = make(map[string]bool, len(group.Skills))
		for _, id := range group.Skills {
			m.editorChosen[id] = true
		}
		m.editorIndex = 0
		m.screen = ScreenGroupEditor
		return
	}
}

func (m *Model) saveEditor() {
	store := group.New(m.paths.Groups)
	selected := make([]string, 0)
	for _, id := range m.editorSkills {
		if m.editorChosen[id] {
			selected = append(selected, id)
		}
	}

	if len(m.editorGroup.Skills) > 0 {
		if _, err := store.Remove(m.editorGroup.Name, m.editorGroup.Skills...); err != nil {
			m.message = err.Error()
			return
		}
	}
	if len(selected) > 0 {
		if _, err := store.Add(m.editorGroup.Name, selected...); err != nil {
			m.message = err.Error()
			return
		}
	}
	if err := m.refresh(); err != nil {
		m.message = err.Error()
		return
	}
	m.screen = ScreenGroups
	m.message = fmt.Sprintf("Saved group %s", m.editorGroup.Name)
}

func (m *Model) openDeleteGroup() {
	if m.selectedGroup == "" {
		m.message = "Select a group before deleting"
		return
	}
	m.deleteGroup = m.selectedGroup
	m.modal = modalDeleteGroup
}

func (m *Model) viewMain() string {
	header := m.viewHeader()
	bodyHeight := m.height - 5
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	groupWidth := m.width / 4
	if groupWidth < 22 {
		groupWidth = 22
	}
	skillsWidth := m.width * 3 / 8
	if skillsWidth < 30 {
		skillsWidth = 30
	}
	detailsWidth := m.width - groupWidth - skillsWidth
	showDetails := detailsWidth >= 24 && m.width >= 90
	if showDetails {
		panels := []string{
			m.panel("Groups", m.viewGroupPanel(), groupWidth, bodyHeight, m.focus == FocusGroups),
			m.panel("Skills", m.viewSkillsPanel(), skillsWidth, bodyHeight, m.focus == FocusSkills),
			m.panel("Details", m.viewDetailsPanel(), detailsWidth, bodyHeight, m.focus == FocusDetails),
		}
		return strings.Join([]string{header, lipgloss.JoinHorizontal(lipgloss.Top, panels...), m.viewFooter()}, "\n")
	}
	if m.detailExpanded {
		return strings.Join([]string{header, m.panel("Details", m.viewDetailsPanel(), m.width, bodyHeight, true), m.viewFooter()}, "\n")
	}

	usableSkillsWidth := m.width - groupWidth
	return strings.Join([]string{
		header,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.panel("Groups", m.viewGroupPanel(), groupWidth, bodyHeight, m.focus == FocusGroups),
			m.panel("Skills", m.viewSkillsPanel(), usableSkillsWidth, bodyHeight, m.focus == FocusSkills),
		),
		m.viewFooter(),
	}, "\n")
}

func (m *Model) viewHeader() string {
	groupName := "All"
	if m.selectedGroup != "" {
		groupName = m.selectedGroup
	}
	header := fmt.Sprintf(
		"Skiller  Installed %d  Active %d  Disabled %d  Conflict %d  Broken %d  Invalid %d  Group: %s",
		m.summary.Installed,
		m.summary.Active,
		m.summary.Disabled,
		m.summary.Conflict,
		m.summary.Broken,
		m.summary.Invalid,
		groupName,
	)
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(header)
}

func (m *Model) viewGroupPanel() string {
	lines := []string{"All  " + fmt.Sprintf("%d", len(m.skills))}
	for _, group := range m.groups {
		lines = append(lines, fmt.Sprintf("%s  %d", group.Name, len(group.Skills)))
	}
	selected := 0
	if m.selectedGroup != "" {
		for index, group := range m.groups {
			if group.Name == m.selectedGroup {
				selected = index + 1
				break
			}
		}
	}
	for index := range lines {
		prefix := "  "
		if index == selected {
			prefix = "> "
		}
		lines[index] = prefix + lines[index]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewSkillsPanel() string {
	visible := m.visibleSkills()
	if len(visible) == 0 {
		return "No matching skills"
	}
	lines := make([]string, 0, len(visible))
	for _, skill := range visible {
		prefix := "  "
		if skill.ID == m.selectedSkill {
			prefix = "> "
		}
		lines = append(lines, prefix+stateLine(skill))
	}
	if m.searchActive {
		lines = append([]string{"Search: " + m.search}, lines...)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewDetailsPanel() string {
	skill, ok := m.selectedSkillValue()
	if !ok {
		return "No skill selected"
	}
	groups := make([]string, 0)
	for _, group := range m.groups {
		for _, id := range group.Skills {
			if id == skill.ID {
				groups = append(groups, group.Name)
				break
			}
		}
	}
	if len(groups) == 0 {
		groups = []string{"-"}
	}
	lines := []string{
		"Name: " + displayName(skill),
		"ID: " + skill.ID,
		"Description: " + valueOrDash(skill.Description),
		"Status: " + stateLine(skill),
		"Groups: " + strings.Join(groups, ", "),
		"Active path: " + valueOrDash(skill.ActivePath),
		"Disabled path: " + valueOrDash(skill.DisabledPath),
		"SKILL.md: " + valueOrDash(skill.SkillFile),
		"Source: " + sourceLine(skill),
	}
	if skill.ActivePath != "" {
		lines = append(lines, "Active source: "+skill.ActiveSource.String())
	}
	if skill.DisabledPath != "" {
		lines = append(lines, "Disabled source: "+skill.DisabledSource.String())
	}
	if skill.ActiveLinkTarget != "" {
		lines = append(lines, "Active target: "+skill.ActiveLinkTarget)
	}
	if skill.DisabledLinkTarget != "" {
		lines = append(lines, "Disabled target: "+skill.DisabledLinkTarget)
	}
	if skill.ActiveIssue != "" {
		lines = append(lines, "Active issue: "+skill.ActiveIssue)
	}
	if skill.DisabledIssue != "" {
		lines = append(lines, "Disabled issue: "+skill.DisabledIssue)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewFooter() string {
	footer := "↑↓/jk move  tab focus  space toggle  / search  g groups  u use  enter details  d doctor  ? help  q quit"
	if m.message != "" {
		footer += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(m.message)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(footer)
}

func (m *Model) viewGroups() string {
	lines := []string{"Groups", ""}
	for _, group := range m.groups {
		prefix := "  "
		if group.Name == m.selectedGroup {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%-32s %d skills", prefix, group.Name, len(group.Skills)))
		if len(group.Missing) > 0 {
			lines = append(lines, "    missing: "+strings.Join(group.Missing, ", "))
		}
	}
	if len(m.groups) == 0 {
		lines = append(lines, "  No groups")
	}
	body := strings.Join(lines, "\n")
	if m.modal == modalGroupName {
		body += "\n\nNew group name: " + m.input + "_"
	}
	if m.modal == modalDeleteGroup {
		body += "\n\nDelete group " + m.deleteGroup + "? enter/y confirm, esc/n cancel"
	}
	if m.message != "" {
		body += "\n\n" + m.message
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Groups"),
		m.panel("Groups", body, m.width, m.height-3, true),
		"n new  e edit  d delete  enter inspect  esc back",
	}, "\n")
}

func (m *Model) viewGroupEditor() string {
	lines := []string{"Edit Group: " + m.editorGroup.Name, ""}
	for index, id := range m.editorSkills {
		mark := "[ ]"
		if m.editorChosen[id] {
			mark = "[x]"
		}
		prefix := "  "
		if index == m.editorIndex {
			prefix = "> "
		}
		lines = append(lines, prefix+mark+" "+id)
	}
	if len(m.editorSkills) == 0 {
		lines = append(lines, "  No installed skills")
	}
	if m.message != "" {
		lines = append(lines, "", m.message)
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Edit Group"),
		m.panel("Edit", strings.Join(lines, "\n"), m.width, m.height-3, true),
		"space toggle  s save  esc cancel",
	}, "\n")
}

func (m *Model) viewGroupDetails() string {
	for _, group := range m.groups {
		if group.Name != m.selectedGroup {
			continue
		}
		lines := []string{"Name: " + group.Name, "Skills:"}
		for _, id := range group.Skills {
			lines = append(lines, "  "+id)
		}
		if len(group.Missing) == 0 {
			lines = append(lines, "Missing: none")
		} else {
			lines = append(lines, "Missing: "+strings.Join(group.Missing, ", "))
		}
		return strings.Join([]string{
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Group Details"),
			m.panel("Group", strings.Join(lines, "\n"), m.width, m.height-3, true),
			"esc back",
		}, "\n")
	}
	return "Group not found\nesc back\n"
}

func (m *Model) viewReconcileModal() string {
	lines := []string{"Activate Group: " + m.plan.Group, "", "Enable"}
	for _, id := range m.plan.Enable {
		lines = append(lines, "  + "+id)
	}
	lines = append(lines, "Disable")
	for _, id := range m.plan.Disable {
		lines = append(lines, "  - "+id)
	}
	lines = append(lines, "Keep")
	for _, id := range m.plan.Keep {
		lines = append(lines, "  = "+id)
	}
	if len(m.plan.Missing) > 0 {
		lines = append(lines, "Missing")
		for _, id := range m.plan.Missing {
			lines = append(lines, "  ? "+id)
		}
	}
	if len(m.plan.Issues) > 0 {
		lines = append(lines, "Issues")
		for _, issue := range m.plan.Issues {
			lines = append(lines, "  ! "+issue)
		}
	}
	lines = append(lines, "", fmt.Sprintf("%d enable  %d disable  %d unchanged", len(m.plan.Enable), len(m.plan.Disable), len(m.plan.Keep)))
	if m.plan.HasIssues() {
		lines = append(lines, "Cannot apply until issues are resolved")
	}

	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Reconcile"),
		m.panel("Activate", strings.Join(lines, "\n"), m.width, m.height-3, true),
		"enter apply  esc cancel",
	}, "\n")
}

func (m *Model) viewDoctor() string {
	lines := []string{"Doctor"}
	for _, check := range m.doctorReport.Checks {
		lines = append(lines, fmt.Sprintf("%s %s: %s", check.Level.Symbol(), check.Name, check.Detail))
	}
	lines = append(lines, "", "Status: "+m.doctorReport.Overall.String())
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Doctor"),
		m.panel("Doctor", strings.Join(lines, "\n"), m.width, m.height-3, true),
		"esc back",
	}, "\n")
}

func (m *Model) viewHelp() string {
	lines := []string{
		"↑/k and ↓/j  move selection",
		"tab          switch panel focus",
		"space        toggle selected skill",
		"/            search by ID, metadata, or group",
		"g            group management",
		"u            preview and apply selected group",
		"enter        expand details",
		"d            doctor/status",
		"esc          close or go back",
		"q            quit",
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Help"),
		m.panel("Keyboard", strings.Join(lines, "\n"), m.width, m.height-3, true),
		"esc back",
	}, "\n")
}

func (m *Model) panel(title, content string, width, height int, focused bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	style := lipgloss.NewStyle().Width(width-2).Height(height-2).Padding(0, 1)
	if focused {
		style = style.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("6"))
	} else {
		style = style.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8"))
	}
	return style.Render(lipgloss.NewStyle().Bold(true).Render(title) + "\n" + content)
}

func stateLine(skill catalog.Skill) string {
	icon := "?"
	switch skill.State {
	case catalog.StateActive:
		icon = "●"
	case catalog.StateDisabled:
		icon = "○"
	case catalog.StateConflict:
		icon = "!"
	case catalog.StateBroken:
		icon = "×"
	case catalog.StateInvalid:
		icon = "?"
	}
	name := displayName(skill)
	if name != skill.ID {
		name = fmt.Sprintf("%s [%s]", name, skill.ID)
	}
	return fmt.Sprintf("%s %s (%s)", icon, name, skill.State)
}

func sourceLine(skill catalog.Skill) string {
	if skill.ActivePath != "" {
		return skill.ActiveSource.String()
	}
	return skill.DisabledSource.String()
}

func displayName(skill catalog.Skill) string {
	if skill.Name == "" {
		return skill.ID
	}
	return skill.Name
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
