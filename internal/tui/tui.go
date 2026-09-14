package tui

import (
	"fmt"
	"slices"
	"strings"

	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/Kklyee/skiller/internal/transaction"
	"github.com/Kklyee/skiller/internal/visibility"
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
)

type modal uint8

const (
	modalNone modal = iota
	modalReconcile
	modalDeleteGroup
	modalGroupName
)

type messageKind uint8

const (
	messageInfo messageKind = iota + 1
	messageSuccess
	messageError
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
	activeGroup    string
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
	messageLevel messageKind
	width        int
	height       int
}

type mainColumn struct {
	title   string
	content string
	width   int
	focused bool
}

type keyHint struct {
	key         string
	description string
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

	_, err = bubbletea.NewProgram(&model).Run()
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
	case bubbletea.KeyPressMsg:
		return m, m.updateKey(message)
	}

	return m, nil
}

func (m *Model) View() bubbletea.View {
	view := bubbletea.NewView(m.viewContent())
	view.AltScreen = true
	return view
}

func (m *Model) viewContent() string {
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
	m.activeGroup = findActiveGroup(groups, skills)
	m.summary = catalog.Summarize(skills)
	m.summary.ActiveDir = m.paths.Active
	m.summary.DisabledDir = m.paths.Disabled
	m.selectedGroup = previousGroup
	m.selectedSkill = previousSkill
	m.normalizeSelection()

	return nil
}

func (m *Model) updateKey(message bubbletea.KeyPressMsg) bubbletea.Cmd {
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
			m.clearMessage()
		}
		return nil
	}

	if key == "ctrl+c" || key == "q" {
		return bubbletea.Quit
	}

	switch key {
	case "tab":
		m.focus++
		if m.focus > FocusSkills {
			m.focus = FocusGroups
		}
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "space":
		if m.focus == FocusSkills {
			m.toggleSelectedSkill()
		}
	case "a":
		if m.focus == FocusSkills {
			m.toggleAllSkills()
		}
	case "/":
		m.searchActive = true
		m.search = ""
	case "enter":
		if m.focus == FocusGroups {
			m.screen = ScreenGroups
			m.detailExpanded = false
			m.clearMessage()
		} else {
			m.detailExpanded = true
		}
	case "esc":
		m.detailExpanded = false
		m.focus = FocusSkills
	case "g":
		m.screen = ScreenGroups
		m.clearMessage()
	case "u":
		m.openReconcile()
	case "d":
		m.doctorReport = doctor.Inspect(m.paths)
		m.screen = ScreenDoctor
	case "?":
		m.screen = ScreenHelp
		m.clearMessage()
	}

	return nil
}

func (m *Model) updateSearch(message bubbletea.KeyPressMsg) bubbletea.Cmd {
	switch message.String() {
	case "esc":
		m.searchActive = false
		m.search = ""
	case "enter":
		m.searchActive = false
	case "backspace", "delete":
		runes := []rune(m.search)
		if len(runes) > 0 {
			m.search = string(runes[:len(runes)-1])
		}
	default:
		m.search += message.Text
	}
	m.normalizeSelection()
	return nil
}

func (m *Model) updateGroups(message bubbletea.KeyPressMsg, key string) bubbletea.Cmd {
	switch key {
	case "ctrl+c", "q":
		return bubbletea.Quit
	case "esc":
		m.screen = ScreenMain
		m.clearMessage()
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
	case "u":
		m.openReconcile()
	case "enter":
		if m.selectedGroup != "" {
			m.screen = ScreenGroupDetails
		}
	}
	return nil
}

func (m *Model) updateEditor(message bubbletea.KeyPressMsg) bubbletea.Cmd {
	key := message.String()
	switch key {
	case "esc":
		m.screen = ScreenGroups
		m.clearMessage()
	case "up", "k":
		if m.editorIndex > 0 {
			m.editorIndex--
		}
	case "down", "j":
		if m.editorIndex+1 < len(m.editorSkills) {
			m.editorIndex++
		}
	case "space":
		if len(m.editorSkills) > 0 {
			id := m.editorSkills[m.editorIndex]
			m.editorChosen[id] = !m.editorChosen[id]
		}
	case "a":
		for _, id := range m.editorSkills {
			m.editorChosen[id] = true
		}
	case "enter":
		m.saveEditor()
	}
	return nil
}

func (m *Model) updateModal(message bubbletea.KeyPressMsg) bubbletea.Cmd {
	key := message.String()
	if m.modal == modalReconcile {
		if key == "esc" {
			m.modal = modalNone
			m.clearMessage()
			return nil
		}
		if key == "enter" {
			if m.plan.HasIssues() {
				m.setMessage(messageInfo, "Cannot apply: resolve missing skills or catalog issues first")
				return nil
			}
			if err := transaction.Apply(m.paths, m.plan); err != nil {
				m.setError(err)
				return nil
			} else if err := m.refresh(); err != nil {
				m.setError(err)
				return nil
			} else {
				m.setMessage(messageSuccess, fmt.Sprintf("Applied group %s", m.plan.Group))
			}
			m.modal = modalNone
		}
		return nil
	}

	if m.modal == modalGroupName {
		switch message.String() {
		case "esc":
			m.modal = modalNone
			m.clearMessage()
		case "enter":
			name := strings.TrimSpace(m.input)
			if name == "" {
				m.setMessage(messageInfo, "Group name is required")
				return nil
			}
			if _, err := group.New(m.paths.Groups).Create(name); err != nil {
				m.setError(err)
				return nil
			} else if err := m.refresh(); err != nil {
				m.setError(err)
				return nil
			} else {
				m.selectedGroup = name
				m.setMessage(messageSuccess, fmt.Sprintf("Created group %s; select skills", name))
				m.openEditor()
			}
			m.modal = modalNone
		case "backspace", "delete":
			runes := []rune(m.input)
			if len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		default:
			m.input += message.Text
		}
		return nil
	}

	if key == "esc" || key == "n" {
		m.modal = modalNone
		m.clearMessage()
		return nil
	}
	if key == "enter" || strings.EqualFold(key, "y") {
		if err := group.New(m.paths.Groups).Delete(m.deleteGroup); err != nil {
			m.setError(err)
		} else if err := m.refresh(); err != nil {
			m.setError(err)
		} else {
			m.selectedGroup = ""
			m.setMessage(messageSuccess, fmt.Sprintf("Deleted group %s", m.deleteGroup))
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
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.setMessage(messageSuccess, fmt.Sprintf("Toggled %s", skill.ID))
}

func (m *Model) toggleAllSkills() {
	skills := m.visibleSkills()
	if len(skills) == 0 {
		return
	}

	allActive := true
	desired := make([]string, 0, len(skills))
	for _, skill := range skills {
		desired = append(desired, skill.ID)
		if skill.State != catalog.StateActive {
			allActive = false
		}
	}
	if allActive {
		desired = nil
	}

	plan := reconcile.Build(group.Group{Name: "all skills", Skills: desired}, skills)
	if plan.HasIssues() {
		m.setMessage(messageError, "Cannot toggle all skills: resolve catalog issues first")
		return
	}
	if plan.Changes() == 0 {
		return
	}
	if err := transaction.Apply(m.paths, plan); err != nil {
		m.setError(err)
		return
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}

	if allActive {
		m.setMessage(messageSuccess, "Disabled all visible skills")
	} else {
		m.setMessage(messageSuccess, "Activated all visible skills")
	}
}

func (m *Model) openReconcile() {
	if m.selectedGroup == "" {
		m.setMessage(messageInfo, "Select a group before pressing u")
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
		m.setMessage(messageInfo, "Select a group before editing")
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
			m.setError(err)
			return
		}
	}
	if len(selected) > 0 {
		if _, err := store.Add(m.editorGroup.Name, selected...); err != nil {
			m.setError(err)
			return
		}
	}
	if err := m.refresh(); err != nil {
		m.setError(err)
		return
	}
	m.screen = ScreenGroups
	m.setMessage(messageSuccess, fmt.Sprintf("Saved group %s", m.editorGroup.Name))
}

func (m *Model) openDeleteGroup() {
	if m.selectedGroup == "" {
		m.setMessage(messageInfo, "Select a group before deleting")
		return
	}
	m.deleteGroup = m.selectedGroup
	m.modal = modalDeleteGroup
}

func (m *Model) viewMain() string {
	header := m.viewHeader()
	bodyHeight := m.height - 7
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	groupWidth, skillsWidth, detailsWidth := mainColumnWidths(m.width)
	showDetails := detailsWidth >= 24 && m.width >= 90
	if showDetails {
		return strings.Join([]string{
			header,
			m.mainColumns([]mainColumn{
				{title: "Groups", content: m.viewGroupPanel(), width: groupWidth, focused: m.focus == FocusGroups},
				{title: "Skills", content: m.viewSkillsPanel(), width: skillsWidth, focused: m.focus == FocusSkills},
				{title: "Details", content: m.viewDetailsPanel(), width: detailsWidth},
			}, bodyHeight),
			m.viewFooter(),
		}, "\n")
	}
	if m.detailExpanded {
		return strings.Join([]string{
			header,
			m.mainColumns([]mainColumn{{title: "Details", content: m.viewDetailsPanel(), width: m.width}}, bodyHeight),
			m.viewFooter(),
		}, "\n")
	}

	usableSkillsWidth := m.width - groupWidth
	return strings.Join([]string{
		header,
		m.mainColumns([]mainColumn{
			{title: "Groups", content: m.viewGroupPanel(), width: groupWidth, focused: m.focus == FocusGroups},
			{title: "Skills", content: m.viewSkillsPanel(), width: usableSkillsWidth, focused: m.focus == FocusSkills},
		}, bodyHeight),
		m.viewFooter(),
	}, "\n")
}

func mainColumnWidths(width int) (groupWidth, skillsWidth, detailsWidth int) {
	groupWidth = width / 5
	if groupWidth < 22 {
		groupWidth = 22
	}
	if groupWidth > 30 {
		groupWidth = 30
	}

	skillsWidth = width / 3
	if skillsWidth < 30 {
		skillsWidth = 30
	}
	if skillsWidth > 60 {
		skillsWidth = 60
	}

	detailsWidth = width - groupWidth - skillsWidth
	return groupWidth, skillsWidth, detailsWidth
}

func (m *Model) mainColumns(columns []mainColumn, height int) string {
	headers := make([]string, 0, len(columns))
	rule := make([]string, 0, len(columns))
	bodies := make([]string, 0, len(columns))
	for _, column := range columns {
		headers = append(headers, mainColumnHeader(column.title, column.width, column.focused))
		rule = append(rule, lipgloss.NewStyle().Width(column.width).Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", column.width)))
		bodies = append(bodies, mainColumnBody(column, height))
	}

	return strings.Join([]string{
		lipgloss.JoinHorizontal(lipgloss.Top, headers...),
		lipgloss.JoinHorizontal(lipgloss.Top, rule...),
		lipgloss.JoinHorizontal(lipgloss.Top, bodies...),
	}, "\n")
}

func mainColumnBody(column mainColumn, height int) string {
	style := lipgloss.NewStyle().Width(column.width).Height(height)
	if column.focused {
		contentWidth := column.width - 2
		if contentWidth < 1 {
			contentWidth = 1
		}
		contentHeight := height - 2
		if contentHeight < 1 {
			contentHeight = 1
		}
		style = lipgloss.NewStyle().
			Width(contentWidth).
			Height(contentHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6"))
	}
	return style.Render(column.content)
}

func mainColumnHeader(title string, width int, focused bool) string {
	style := lipgloss.NewStyle().Width(width).Bold(true).Foreground(lipgloss.Color("8"))
	if focused {
		style = style.Foreground(lipgloss.Color("6")).Underline(true)
	}
	return style.Render(title)
}

func (m *Model) viewHeader() string {
	groupName := "All"
	if m.selectedGroup != "" {
		groupName = m.selectedGroup
	}
	parts := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller"),
		headerMetric("Installed", m.summary.Installed, "6"),
		headerMetric("Active", m.summary.Active, "10"),
		headerMetric("Disabled", m.summary.Disabled, "8"),
		headerMetric("Conflict", m.summary.Conflict, "9"),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Group: " + groupName),
	}
	return strings.Join(parts, "  ")
}

func headerMetric(label string, value int, color string) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	if value == 0 && label != "Installed" {
		style = style.Foreground(lipgloss.Color("8"))
	}
	return style.Render(fmt.Sprintf("%s %d", label, value))
}

func (m *Model) viewGroupPanel() string {
	lines := []string{"All  " + fmt.Sprintf("%d", len(m.skills))}
	for _, group := range m.groups {
		marker := "  "
		if group.Name == m.activeGroup {
			marker = stateStyle(catalog.StateActive).Render("●") + " "
		}
		lines = append(lines, fmt.Sprintf("%s%s  %d", marker, group.Name, len(group.Skills)))
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

func findActiveGroup(groups []group.Group, skills []catalog.Skill) string {
	active := make(map[string]struct{})
	for _, skill := range skills {
		switch skill.State {
		case catalog.StateActive:
			active[skill.ID] = struct{}{}
		case catalog.StateConflict, catalog.StateBroken, catalog.StateInvalid:
			return ""
		}
	}

	matches := make([]string, 0, 1)
	for _, candidate := range groups {
		if len(candidate.Missing) > 0 || len(candidate.Skills) != len(active) {
			continue
		}
		matched := true
		for _, id := range candidate.Skills {
			if _, ok := active[id]; !ok {
				matched = false
				break
			}
		}
		if matched {
			matches = append(matches, candidate.Name)
		}
	}
	if len(matches) != 1 {
		return ""
	}
	return matches[0]
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
		"Description: " + valueOrDash(skill.Description),
		"Status: " + stateStyle(skill.State).Render(skill.State.String()),
		"Groups: " + strings.Join(groups, ", "),
	}
	return strings.Join(lines, "\n")
}

func (m *Model) setMessage(kind messageKind, message string) {
	m.message = message
	m.messageLevel = kind
}

func (m *Model) setError(err error) {
	m.setMessage(messageError, err.Error())
}

func (m *Model) clearMessage() {
	m.message = ""
	m.messageLevel = 0
}

func (m *Model) renderedMessage() string {
	if m.message == "" {
		return ""
	}
	return messageStyle(m.messageLevel).Render(m.message)
}

func messageStyle(kind messageKind) lipgloss.Style {
	color := lipgloss.Color("11")
	switch kind {
	case messageSuccess:
		color = lipgloss.Color("10")
	case messageError:
		color = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func (m *Model) viewFooter() string {
	hints := []keyHint{
		{key: "↑↓/jk", description: "move"},
		{key: "tab", description: "focus"},
	}
	if m.focus == FocusSkills {
		hints = append(hints,
			keyHint{key: "space", description: "toggle"},
			keyHint{key: "a", description: "all"},
		)
	}
	hints = append(hints,
		keyHint{key: "/", description: "search"},
		keyHint{key: "g", description: "groups"},
		keyHint{key: "u", description: "use"},
	)
	if m.focus == FocusGroups {
		hints = append(hints, keyHint{key: "enter", description: "groups"})
	} else {
		hints = append(hints, keyHint{key: "enter", description: "details"})
	}
	hints = append(hints,
		keyHint{key: "d", description: "doctor"},
		keyHint{key: "?", description: "help"},
		keyHint{key: "q", description: "quit"},
	)
	if m.searchActive {
		hints = []keyHint{
			{key: "type", description: "filter"},
			{key: "enter", description: "finish search"},
			{key: "esc", description: "cancel"},
		}
	} else if m.detailExpanded {
		hints = []keyHint{
			{key: "↑↓/jk", description: "move"},
			{key: "space", description: "toggle"},
			{key: "esc", description: "collapse"},
		}
	}
	lines := []string{renderKeyHints(hints...)}
	if m.message != "" {
		lines = append(lines, messageStyle(m.messageLevel).Render(m.message))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewGroups() string {
	body := m.viewGroupManagerList()
	details := m.viewGroupManagerDetails()
	leftWidth := m.width / 4
	if leftWidth < 28 {
		leftWidth = 28
	}
	if leftWidth > 36 {
		leftWidth = 36
	}
	if leftWidth > m.width-24 {
		leftWidth = m.width - 24
	}
	if leftWidth < 4 {
		leftWidth = 4
	}
	rightWidth := m.width - leftWidth
	panelHeight := m.height - 3
	if panelHeight < 3 {
		panelHeight = 3
	}
	footer := renderKeyHints(
		keyHint{key: "↑↓/jk", description: "move"},
		keyHint{key: "n", description: "new"},
		keyHint{key: "e", description: "edit"},
		keyHint{key: "d", description: "delete"},
		keyHint{key: "u", description: "use"},
		keyHint{key: "enter", description: "inspect"},
		keyHint{key: "esc", description: "back"},
	)
	if m.modal == modalGroupName {
		footer = renderKeyHints(
			keyHint{key: "type", description: "name"},
			keyHint{key: "enter", description: "create"},
			keyHint{key: "esc", description: "cancel"},
		)
	} else if m.modal == modalDeleteGroup {
		footer = renderKeyHints(
			keyHint{key: "enter/y", description: "delete"},
			keyHint{key: "esc/n", description: "cancel"},
		)
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Groups"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.panel("Groups", body, leftWidth, panelHeight, true),
			m.panel(m.groupManagerTitle(), details, rightWidth, panelHeight, false),
		),
		footer,
	}, "\n")
}

func (m *Model) viewGroupManagerList() string {
	lines := make([]string, 0, len(m.groups)+4)
	for _, group := range m.groups {
		prefix := "  "
		if group.Name == m.selectedGroup {
			prefix = "> "
		}
		marker := "  "
		if group.Name == m.activeGroup {
			marker = stateStyle(catalog.StateActive).Render("●") + " "
		}
		lines = append(lines, prefix+marker+group.Name+fmt.Sprintf("  %d", len(group.Skills)))
		if len(group.Missing) > 0 {
			lines = append(lines, helpTextStyle().Render("    missing: "+strings.Join(group.Missing, ", ")))
		}
	}
	if len(m.groups) == 0 {
		lines = append(lines, "  No groups")
	}
	if m.modal == modalGroupName {
		lines = append(lines, "", "New group name: "+m.input+"_")
	}
	if m.modal == modalDeleteGroup {
		lines = append(lines, "", "Delete group "+m.deleteGroup+"? enter/y confirm, esc/n cancel")
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join(lines, "\n")
}

func (m *Model) groupManagerTitle() string {
	if group, ok := m.selectedGroupValue(); ok {
		return group.Name
	}
	return "Details"
}

func (m *Model) viewGroupManagerDetails() string {
	group, ok := m.selectedGroupValue()
	if !ok {
		return helpTextStyle().Render("Select a group to inspect its members")
	}

	status := helpTextStyle().Render("○ Not applied")
	if group.Name == m.activeGroup {
		status = stateStyle(catalog.StateActive).Render("● Applied")
	}
	lines := []string{status, "", helpTextStyle().Bold(true).Render("Members")}
	if len(group.Skills) == 0 {
		lines = append(lines, helpTextStyle().Render("  No skills"))
	} else {
		installed := make(map[string]catalog.Skill, len(m.skills))
		for _, skill := range m.skills {
			installed[skill.ID] = skill
		}
		for _, id := range group.Skills {
			skill, ok := installed[id]
			if !ok {
				lines = append(lines, messageStyle(messageInfo).Render("? ")+id+" (missing)")
				continue
			}
			lines = append(lines, stateStyle(skill.State).Render(stateIcon(skill.State))+" "+displayName(skill))
		}
	}

	plan := reconcile.Build(group, m.skills)
	lines = append(lines,
		"",
		helpTextStyle().Bold(true).Render("Activation Preview"),
		fmt.Sprintf("Keep %d   Enable %d   Disable %d", len(plan.Keep), len(plan.Enable), len(plan.Disable)),
	)
	if plan.HasIssues() {
		lines = append(lines, messageStyle(messageError).Render("! Resolve issues before use"))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) selectedGroupValue() (group.Group, bool) {
	for _, candidate := range m.groups {
		if candidate.Name == m.selectedGroup {
			return candidate, true
		}
	}
	return group.Group{}, false
}

func (m *Model) viewGroupEditor() string {
	lines := []string{"Edit Group: " + m.editorGroup.Name, ""}
	for index, id := range m.editorSkills {
		mark := editorUnselectedStyle().Render("[ ]")
		if m.editorChosen[id] {
			mark = editorSelectedStyle().Render("[x]")
		}
		prefix := "  "
		if index == m.editorIndex {
			prefix = editorCursorStyle().Render("> ")
		}
		lines = append(lines, prefix+mark+" "+id)
	}
	if len(m.editorSkills) == 0 {
		lines = append(lines, "  No installed skills")
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Edit Group"),
		m.panel("Edit", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "↑↓/jk", description: "move"},
			keyHint{key: "space", description: "select"},
			keyHint{key: "a", description: "select all"},
			keyHint{key: "enter", description: "save"},
			keyHint{key: "esc", description: "cancel"},
		),
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
			renderKeyHints(keyHint{key: "esc", description: "back"}),
		}, "\n")
	}
	return "Group not found\nesc back\n"
}

func (m *Model) viewReconcileModal() string {
	lines := []string{"Activate Group: " + m.plan.Group, "", messageStyle(messageSuccess).Render("Enable")}
	for _, id := range m.plan.Enable {
		lines = append(lines, messageStyle(messageSuccess).Render("  +")+" "+id)
	}
	lines = append(lines, messageStyle(messageError).Render("Disable"))
	for _, id := range m.plan.Disable {
		lines = append(lines, messageStyle(messageError).Render("  -")+" "+id)
	}
	lines = append(lines, messageStyle(messageInfo).Render("Keep"))
	for _, id := range m.plan.Keep {
		lines = append(lines, helpTextStyle().Render("  =")+" "+id)
	}
	if len(m.plan.Missing) > 0 {
		lines = append(lines, messageStyle(messageInfo).Render("Missing"))
		for _, id := range m.plan.Missing {
			lines = append(lines, messageStyle(messageInfo).Render("  ?")+" "+id)
		}
	}
	if len(m.plan.Issues) > 0 {
		lines = append(lines, messageStyle(messageError).Render("Issues"))
		for _, issue := range m.plan.Issues {
			lines = append(lines, messageStyle(messageError).Render("  !")+" "+issue)
		}
	}
	lines = append(lines, "", fmt.Sprintf("%d enable  %d disable  %d unchanged", len(m.plan.Enable), len(m.plan.Disable), len(m.plan.Keep)))
	if m.plan.HasIssues() {
		lines = append(lines, "Cannot apply until issues are resolved")
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}

	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Reconcile"),
		m.panel("Activate", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(keyHint{key: "enter", description: "apply"}, keyHint{key: "esc", description: "cancel"}),
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
		renderKeyHints(keyHint{key: "esc", description: "back"}),
	}, "\n")
}

func (m *Model) viewHelp() string {
	lines := []string{
		helpLine("↑/k ↓/j", "move selection"),
		helpLine("tab", "switch panel focus"),
		helpLine("space", "toggle selected skill"),
		helpLine("a", "toggle all visible skills"),
		helpLine("/", "search by ID, metadata, or group"),
		helpLine("g", "group management"),
		helpLine("u", "preview and apply selected group"),
		helpLine("enter", "expand details"),
		helpLine("d", "doctor/status"),
		helpLine("esc", "close or go back"),
		helpLine("q", "quit"),
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Help"),
		m.panel("Keyboard", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(keyHint{key: "esc/q", description: "back"}),
	}, "\n")
}

func renderKeyHints(hints ...keyHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, helpKeyStyle().Render(hint.key)+" "+helpTextStyle().Render(hint.description))
	}
	return strings.Join(parts, "  ")
}

func helpLine(key, description string) string {
	return fmt.Sprintf("  %s %s", helpKeyStyle().Render(fmt.Sprintf("%-12s", key)), helpTextStyle().Render(description))
}

func helpKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
}

func helpTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func editorSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
}

func editorUnselectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func editorCursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
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
	name := displayName(skill)
	if name != skill.ID {
		name = fmt.Sprintf("%s [%s]", name, skill.ID)
	}
	return fmt.Sprintf("%s %s", stateStyle(skill.State).Render(stateIcon(skill.State)), name)
}

func stateStyle(state catalog.State) lipgloss.Style {
	color := lipgloss.Color("8")
	switch state {
	case catalog.StateActive:
		color = lipgloss.Color("10")
	case catalog.StateDisabled:
		color = lipgloss.Color("8")
	case catalog.StateConflict, catalog.StateBroken:
		color = lipgloss.Color("9")
	case catalog.StateInvalid:
		color = lipgloss.Color("13")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func stateIcon(state catalog.State) string {
	icon := "?"
	switch state {
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
	return icon
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
