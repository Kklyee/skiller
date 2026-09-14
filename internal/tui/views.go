package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/reconcile"
	"strings"
)

func (m *Model) View() bubbletea.View {
	view := bubbletea.NewView(m.viewContent())
	view.AltScreen = true
	return view
}

func (m *Model) viewContent() string {
	if m.width < 60 || m.height < 8 {
		return "Terminal too small. Resize to at least 60x8.\n"
	}
	if m.startupActive {
		return m.viewStartup()
	}
	if m.modal == modalReconcile {
		return m.viewReconcileModal()
	}
	if m.modal == modalGroupName {
		return m.viewGroupNameModal()
	}
	if m.modal == modalDeleteGroup {
		return m.viewDeleteGroupModal()
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

func (m *Model) viewMain() string {
	header := m.viewHeader()
	bodyHeight := m.height - 10
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	groupWidth, skillsWidth, detailsWidth := mainColumnWidths(m.width)
	showDetails := detailsWidth >= 24 && m.width >= 90
	if showDetails {
		return strings.Join([]string{
			m.viewBrand(),
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
			m.viewBrand(),
			header,
			m.mainColumns([]mainColumn{{title: "Details", content: m.viewDetailsPanel(), width: m.width}}, bodyHeight),
			m.viewFooter(),
		}, "\n")
	}

	usableSkillsWidth := m.width - groupWidth
	return strings.Join([]string{
		m.viewBrand(),
		header,
		m.mainColumns([]mainColumn{
			{title: "Groups", content: m.viewGroupPanel(), width: groupWidth, focused: m.focus == FocusGroups},
			{title: "Skills", content: m.viewSkillsPanel(), width: usableSkillsWidth, focused: m.focus == FocusSkills},
		}, bodyHeight),
		m.viewFooter(),
	}, "\n")
}

func (m *Model) viewHeader() string {
	groupName := "All"
	if m.selectedGroup != "" {
		groupName = m.selectedGroup
	}
	parts := []string{
		headerMetric("Installed", m.summary.Installed, "6"),
		headerMetric("Active", m.summary.Active, "10"),
		headerMetric("Disabled", m.summary.Disabled, "8"),
		headerMetric("Conflict", m.summary.Conflict, "9"),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Group: " + groupName),
	}
	return strings.Join(parts, "  ")
}

func (m *Model) viewBrand() string {
	return startupLogo(startupFrameCount - 1)
}

func headerMetric(label string, value int, color string) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	if value == 0 && label != "Installed" {
		style = style.Foreground(lipgloss.Color("8"))
	}
	return style.Render(fmt.Sprintf("%s %d", label, value))
}

func (m *Model) viewGroupPanel() string {
	allMarker := "  "
	if allSkillsActive(m.skills) {
		allMarker = stateStyle(catalog.StateActive).Render("●") + " "
	}
	lines := []string{allMarker + groupNameStyle().Render("All") + "  " + fmt.Sprintf("%d", len(m.skills))}
	for _, group := range m.groups {
		marker := "  "
		if group.Name == m.activeGroup {
			marker = stateStyle(catalog.StateActive).Render("●") + " "
		}
		lines = append(lines, fmt.Sprintf("%s%s  %d", marker, groupNameStyle().Render(group.Name), len(group.Skills)))
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
			prefix = selectedRowStyle().Render("›") + " "
		}
		lines[index] = prefix + lines[index]
	}
	start, end := viewportBounds(len(lines), selected, m.mainPanelContentHeight())
	return strings.Join(lines[start:end], "\n")
}

func allSkillsActive(skills []catalog.Skill) bool {
	if len(skills) == 0 {
		return false
	}
	for _, skill := range skills {
		if skill.State != catalog.StateActive {
			return false
		}
	}
	return true
}

func (m *Model) viewSkillsPanel() string {
	visible := m.visibleSkills()
	lines := make([]string, 0, len(visible)+1)
	contentHeight := m.mainPanelContentHeight()
	if m.searchActive {
		lines = append(lines, m.viewSearchBar(len(visible)))
		contentHeight--
	}
	if len(visible) == 0 {
		lines = append(lines, "No matching skills")
		return strings.Join(lines, "\n")
	}
	selected := 0
	for index, skill := range visible {
		if skill.ID == m.selectedSkill {
			selected = index
			break
		}
	}
	start, end := viewportBounds(len(visible), selected, contentHeight)
	for _, skill := range visible[start:end] {
		lines = append(lines, skillRowWithPin(skill, skill.ID == m.selectedSkill, m.isPinned(skill.ID)))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewSearchBar(matches int) string {
	query := m.search
	queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	if query == "" {
		query = "type to filter"
		queryStyle = helpTextStyle()
	}
	label := "matches"
	if matches == 1 {
		label = "match"
	}
	return selectedRowStyle().Render("/") + " " + queryStyle.Render(query+"▌") + " " + helpTextStyle().Render(fmt.Sprintf("(%d %s)", matches, label))
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
	pinned := "no"
	if m.isPinned(skill.ID) {
		pinned = pinStyle().Render("yes")
	}
	lines = append(lines, "Pinned: "+pinned)
	return strings.Join(lines, "\n")
}

func (m *Model) renderedMessage() string {
	if m.message == "" {
		return ""
	}
	return messageStyle(m.messageLevel).Render(m.message)
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
			prefix = selectedRowStyle().Render("›") + " "
		}
		marker := "  "
		if group.Name == m.activeGroup {
			marker = stateStyle(catalog.StateActive).Render("●") + " "
		}
		name := groupNameStyle().Render(group.Name)
		if group.Name == m.selectedGroup {
			name = groupNameStyle().Bold(true).Render(group.Name)
		}
		lines = append(lines, prefix+marker+name+fmt.Sprintf("  %d", len(group.Skills)))
		if len(group.Missing) > 0 {
			lines = append(lines, helpTextStyle().Render("    missing: "+strings.Join(group.Missing, ", ")))
		}
	}
	if len(m.groups) == 0 {
		lines = append(lines, "  No groups")
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

	plan := reconcile.BuildWithPins(group, m.skills, m.pins)
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

func (m *Model) viewDoctor() string {
	lines := make([]string, 0, len(m.doctorReport.Checks)+5)
	section := ""
	for _, check := range m.doctorReport.Checks {
		if check.Section != section {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, doctorSectionStyle().Render(check.Section))
			section = check.Section
		}
		lines = append(lines, "  "+doctorLevelStyle(check.Level).Render(check.Level.Symbol())+" "+check.Name+"  "+helpTextStyle().Render(check.Detail))
	}
	lines = append(lines, "", "Status: "+doctorLevelStyle(m.doctorReport.Overall).Render(m.doctorReport.Overall.Symbol()+" "+m.doctorReport.Overall.String()))
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

func stateLine(skill catalog.Skill) string {
	return skillRow(skill, false)
}

func skillRow(skill catalog.Skill, selected bool) string {
	return skillRowWithPin(skill, selected, false)
}

func skillRowWithPin(skill catalog.Skill, selected, pinned bool) string {
	name := displayName(skill)
	if name != skill.ID {
		name = fmt.Sprintf("%s [%s]", name, skill.ID)
	}
	if selected {
		name = selectedRowStyle().Render(name)
	}
	row := fmt.Sprintf("%s %s", stateStyle(skill.State).Render(stateIcon(skill.State)), name)
	if pinned {
		row = pinStyle().Render("◆") + " " + row
	}
	return row
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
