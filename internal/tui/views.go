package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/environment"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/profile"
	skillprovenance "github.com/Kklyee/skiller/internal/provenance"
	"github.com/Kklyee/skiller/internal/reconcile"
	"github.com/charmbracelet/x/ansi"
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
	if m.modal == modalProfileName {
		return m.viewProfileNameModal()
	}
	if m.modal == modalDeleteProfile {
		return m.viewDeleteProfileModal()
	}
	if m.modal == modalDeleteSkills {
		return m.viewDeleteSkillsModal()
	}
	if m.modal == modalBatch {
		return m.viewBatchModal()
	}
	if m.modal == modalBatchGroup {
		return m.viewBatchGroupModal()
	}
	if m.modal == modalPalette {
		return m.viewCommandPalette()
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
	case ScreenProfiles:
		return m.viewProfiles()
	case ScreenProfileEditor:
		return m.viewProfileEditor()
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
	skillsTitle := "Skills"
	if selected := m.selectedSkillCount(); selected > 0 {
		skillsTitle = fmt.Sprintf("Skills [%d selected]", selected)
	}
	showDetails := detailsWidth >= 24 && m.width >= 90
	if showDetails {
		return strings.Join([]string{
			m.viewBrand(),
			header,
			m.mainColumns([]mainColumn{
				{title: "Groups", content: m.viewGroupPanel(), width: groupWidth, focused: m.focus == FocusGroups},
				{title: skillsTitle, content: m.viewSkillsPanel(), width: skillsWidth, focused: m.focus == FocusSkills},
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
			{title: skillsTitle, content: m.viewSkillsPanel(), width: usableSkillsWidth, focused: m.focus == FocusSkills},
		}, bodyHeight),
		m.viewFooter(),
	}, "\n")
}

func (m *Model) viewProfiles() string {
	leftWidth := m.width / 3
	if leftWidth < 30 {
		leftWidth = 30
	}
	if leftWidth > 42 {
		leftWidth = 42
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
		keyHint{key: "u/enter", description: "use"},
		keyHint{key: "esc", description: "back"},
	)
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Profiles"),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.panel("Profiles", m.viewProfileList(), leftWidth, panelHeight, true),
			m.panel(m.profileDetailsTitle(), m.viewProfileDetails(), rightWidth, panelHeight, false),
		),
		footer,
	}, "\n")
}

func (m *Model) viewProfileEditor() string {
	sectionNames := []string{"Groups", "Skills", "Exclude"}
	tabs := make([]string, 0, len(sectionNames))
	for index, name := range sectionNames {
		if index == m.profileEditorSection {
			tabs = append(tabs, selectedRowStyle().Render("["+name+"]"))
			continue
		}
		tabs = append(tabs, helpTextStyle().Render(name))
	}
	items := m.profileEditorItems()
	footer := renderKeyHints(
		keyHint{key: "↑↓/jk", description: "move"},
		keyHint{key: "tab", description: "section"},
		keyHint{key: "space", description: "toggle"},
		keyHint{key: "a", description: "select all"},
		keyHint{key: "enter", description: "save"},
		keyHint{key: "esc", description: "cancel"},
	)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Edit Profile")
	panelHeight := m.height - lipgloss.Height(title) - lipgloss.Height(footer) - 2
	if panelHeight < 3 {
		panelHeight = 3
	}
	contentHeight := panelHeight - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	lines := []string{
		strings.Join(tabs, "  "),
		helpTextStyle().Render(fmt.Sprintf("Select %s (%d selected)", sectionNames[m.profileEditorSection], m.profileEditorSelectedCount())),
		"",
	}
	if len(items) == 0 {
		lines = append(lines, helpTextStyle().Render("No available items"))
	} else {
		start, end := viewportBounds(len(items), m.profileEditorIndex, contentHeight)
		for index, id := range items[start:end] {
			itemIndex := start + index
			mark := editorUnselectedStyle().Render("[ ]")
			if m.profileEditorChosen(id) {
				mark = editorSelectedStyle().Render("[x]")
			}
			prefix := "  "
			if itemIndex == m.profileEditorIndex {
				prefix = editorCursorStyle().Render("> ")
			}
			pinMarker := ""
			if m.profileEditorSection != profileEditorGroups && m.isPinned(id) {
				pinMarker = " " + pinStyle().Render("◆")
			}
			lines = append(lines, prefix+mark+" "+id+pinMarker)
		}
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join([]string{
		title,
		m.panel("Edit Profile: "+m.profileEditor.Name, strings.Join(lines, "\n"), m.width, panelHeight, true),
		footer,
	}, "\n")
}

func (m *Model) profileEditorSelectedCount() int {
	selected := m.profileEditorGroups
	if m.profileEditorSection == profileEditorSkills {
		selected = m.profileEditorSkills
	} else if m.profileEditorSection == profileEditorExclude {
		selected = m.profileEditorExclude
	}
	count := 0
	for _, value := range selected {
		if value {
			count++
		}
	}
	return count
}

func (m *Model) viewCommandPalette() string {
	commands := m.filteredPaletteCommands()
	lines := []string{selectedRowStyle().Render(":" + m.paletteQuery + "▌"), ""}
	for index, command := range commands {
		prefix := "  "
		if index == m.paletteIndex {
			prefix = selectedRowStyle().Render("›") + " "
		}
		lines = append(lines, prefix+helpKeyStyle().Render(command.title)+"  "+helpTextStyle().Render(command.description))
	}
	if len(commands) == 0 {
		lines = append(lines, helpTextStyle().Render("No matching commands"))
	}
	width := m.width - 8
	if width < 42 {
		width = 42
	}
	if width > 84 {
		width = 84
	}
	if width > m.width {
		width = m.width
	}
	height := len(lines) + 3
	if height > m.height-3 {
		height = m.height - 3
	}
	if height < 3 {
		height = 3
	}
	panel := m.panel("Command Palette", strings.Join(lines, "\n"), width, height, true)
	footer := renderKeyHints(
		keyHint{key: "↑↓/jk", description: "move"},
		keyHint{key: "enter", description: "run"},
		keyHint{key: "esc", description: "close"},
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, strings.Join([]string{panel, footer}, "\n"))
}

func (m *Model) viewProfileList() string {
	if len(m.profiles) == 0 {
		return helpTextStyle().Render("No profiles")
	}
	lines := make([]string, 0, len(m.profiles))
	for _, stored := range m.profiles {
		prefix := "  "
		if stored.Name == m.selectedProfile {
			prefix = selectedRowStyle().Render("›") + " "
		}
		status := m.statusForTarget(environment.KindProfile, stored.Name)
		marker := environmentStatusStyle(status).Render(environmentStatusIcon(status))
		name := groupNameStyle().Render(stored.Name)
		if stored.Name == m.selectedProfile {
			name = groupNameStyle().Bold(true).Render(stored.Name)
		}
		lines = append(lines, fmt.Sprintf("%s%s %s  %s", prefix, marker, name, helpTextStyle().Render(profileSummary(stored))))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) profileDetailsTitle() string {
	if stored, ok := m.selectedProfileValue(); ok {
		return stored.Name
	}
	return "Details"
}

func (m *Model) viewProfileDetails() string {
	stored, ok := m.selectedProfileValue()
	if !ok {
		return helpTextStyle().Render("Select a profile to inspect its environment")
	}
	target := profile.Resolve(stored, m.groups)
	profileStatus := m.statusForTarget(environment.KindProfile, stored.Name)
	status := environmentStatusStyle(profileStatus).Render(environmentStatusIcon(profileStatus) + " " + environmentStatusLabel(profileStatus))
	lines := []string{
		"Name: " + groupNameStyle().Render(stored.Name),
		"Status: " + status,
		"Groups: " + listOrDash(stored.Groups),
		"Skills: " + listOrDash(stored.Skills),
		"Exclude: " + listOrDash(stored.Exclude),
		"Resolved: " + fmt.Sprintf("%d skills", len(target.Group.Skills)),
	}
	if len(target.MissingGroups) > 0 {
		lines = append(lines, messageStyle(messageError).Render("Missing groups: "+strings.Join(target.MissingGroups, ", ")))
	}
	if missing := profile.MissingSkills(target, m.skills); len(missing) > 0 {
		lines = append(lines, messageStyle(messageError).Render("Missing skills: "+strings.Join(missing, ", ")))
	}
	return strings.Join(lines, "\n")
}

func profileSummary(stored profile.Profile) string {
	return fmt.Sprintf("%d groups  %d skills  %d excluded", len(stored.Groups), len(stored.Skills), len(stored.Exclude))
}

func listOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func (m *Model) viewHeader() string {
	parts := []string{
		headerMetric("Installed", m.summary.Installed, "6"),
		headerMetric("Active", m.summary.Active, "10"),
		headerMetric("Disabled", m.summary.Disabled, "8"),
		headerMetric("Conflict", m.summary.Conflict, "9"),
		m.viewTargetHeader(),
	}
	return strings.Join(parts, "  ")
}

func (m *Model) viewTargetHeader() string {
	if !m.targetLoaded {
		return environmentStatusStyle(environment.StatusManual).Render("Environment: Manual")
	}
	label := "Group"
	name := m.appliedTarget.Name
	switch m.appliedTarget.Kind {
	case environment.KindProfile:
		label = "Profile"
	}
	return environmentStatusStyle(m.targetStatus).Render(label + ": " + name)
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
	allStatus := m.statusForTarget(environment.KindGroup, allGroupName)
	if allStatus != environment.StatusManual {
		allMarker = environmentStatusStyle(allStatus).Render(environmentStatusIcon(allStatus)) + " "
	}
	lines := []string{allMarker + groupNameStyle().Render("All") + "  " + fmt.Sprintf("%d", len(m.skills))}
	for _, group := range m.groups {
		marker := "  "
		status := m.statusForTarget(environment.KindGroup, group.Name)
		if status != environment.StatusManual {
			marker = environmentStatusStyle(status).Render(environmentStatusIcon(status)) + " "
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

func (m *Model) viewSkillsPanel() string {
	visible := m.visibleSkills()
	lines := make([]string, 0, len(visible)+1)
	contentHeight := m.mainPanelContentHeight()
	if toolbar := m.viewSelectionToolbar(); toolbar != "" {
		lines = append(lines, toolbar)
		contentHeight--
	}
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
		lines = append(lines, skillRowWithSelection(skill, skill.ID == m.selectedSkill, m.isPinned(skill.ID), m.selectedSkills[skill.ID]))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewSelectionToolbar() string {
	if m.selectedSkillCount() == 0 {
		return ""
	}
	return selectionStyle().Render(fmt.Sprintf("✓ %d selected", m.selectedSkillCount())) + "  " +
		renderKeyHints(
			keyHint{key: "b", description: "batch"},
			keyHint{key: "delete", description: "remove"},
		)
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
	if origin, ok := m.provenance[skill.ID]; ok {
		if origin.Source == "" {
			origin.Source = sourceLine(skill)
		}
		lines = append(lines,
			"Source: "+valueOrDash(origin.Source),
			"Repository: "+valueOrDash(origin.Repository),
			"Installer: "+valueOrDash(origin.Installer),
			"Revision: "+valueOrDash(origin.Revision),
		)
		if source := updateSource(origin); source != "" {
			lines = append(lines, "Update source: "+source)
		}
	}
	lines = append(lines, diagnosticDetails(skill)...)
	pinned := "no"
	if m.isPinned(skill.ID) {
		pinned = pinStyle().Render("yes")
	}
	lines = append(lines, "Pinned: "+pinned)
	return strings.Join(lines, "\n")
}

func diagnosticDetails(skill catalog.Skill) []string {
	issue := skill.ActiveIssue
	if issue == "" {
		issue = skill.DisabledIssue
	}
	switch skill.State {
	case catalog.StateConflict:
		lines := []string{
			"Issue: " + messageStyle(messageError).Render(valueOrDash(issue)),
			"Active path: " + helpTextStyle().Render(valueOrDash(skill.ActivePath)),
			"Disabled path: " + helpTextStyle().Render(valueOrDash(skill.DisabledPath)),
		}
		return lines
	case catalog.StateBroken, catalog.StateInvalid:
		path := skill.ActivePath
		if path == "" {
			path = skill.DisabledPath
		}
		lines := []string{"Issue: " + messageStyle(messageError).Render(valueOrDash(issue))}
		if path != "" {
			lines = append(lines, "Path: "+helpTextStyle().Render(path))
		}
		if skill.SkillFile != "" {
			lines = append(lines, "SKILL.md: "+helpTextStyle().Render(skill.SkillFile))
		}
		return lines
	default:
		return nil
	}
}

func updateSource(origin skillprovenance.Entry) string {
	if origin.Installer != "" {
		return origin.Installer
	}
	return origin.Repository
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
			keyHint{key: "x", description: "mark"},
			keyHint{key: "b", description: "batch"},
			keyHint{key: "delete", description: "remove"},
		)
	}
	hints = append(hints,
		keyHint{key: "/", description: "search"},
		keyHint{key: "g", description: "groups"},
		keyHint{key: "p", description: "profiles"},
		keyHint{key: ":", description: "commands"},
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
	lines := make([]string, 0, 2)
	if m.selectedSkillCount() > 0 {
		lines = append(lines, selectionStyle().Render(fmt.Sprintf("%d selected", m.selectedSkillCount())))
	}
	lines = append(lines, renderKeyHints(hints...))
	if m.message != "" {
		lines = append(lines, messageStyle(m.messageLevel).Render(m.message))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewGroups() string {
	body := m.viewGroupManagerList()
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
	footer := renderKeyHints(
		keyHint{key: "↑↓/jk", description: "move"},
		keyHint{key: "n", description: "new"},
		keyHint{key: "e", description: "edit"},
		keyHint{key: "d", description: "delete"},
		keyHint{key: "u", description: "use"},
		keyHint{key: "enter", description: "inspect"},
		keyHint{key: "esc", description: "back"},
	)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Groups")
	panelHeight := m.height - lipgloss.Height(title) - lipgloss.Height(footer) - 2
	if panelHeight < 3 {
		panelHeight = 3
	}
	return strings.Join([]string{
		title,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.panel("Groups", body, leftWidth, panelHeight, true),
			m.panel(m.groupManagerTitle(), m.viewGroupManagerDetails(rightWidth-4), rightWidth, panelHeight, false),
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
		status := m.statusForTarget(environment.KindGroup, group.Name)
		if status != environment.StatusManual {
			marker = environmentStatusStyle(status).Render(environmentStatusIcon(status)) + " "
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

func (m *Model) viewGroupManagerDetails(width int) string {
	group, ok := m.selectedGroupValue()
	if !ok {
		return helpTextStyle().Render("Select a group to inspect its members")
	}

	groupStatus := m.statusForTarget(environment.KindGroup, group.Name)
	status := environmentStatusStyle(groupStatus).Render(environmentStatusIcon(groupStatus) + " " + environmentStatusLabel(groupStatus))
	lines := []string{status, "", helpTextStyle().Bold(true).Render("Members")}
	if len(group.Skills) == 0 {
		lines = append(lines, helpTextStyle().Render("  No skills"))
	} else {
		installed := make(map[string]catalog.Skill, len(m.skills))
		for _, skill := range m.skills {
			installed[skill.ID] = skill
		}
		members := make([]string, 0, len(group.Skills))
		for _, id := range group.Skills {
			skill, ok := installed[id]
			if !ok {
				members = append(members, messageStyle(messageInfo).Render("? ")+id+" (missing)")
				continue
			}
			pinMarker := ""
			if m.isPinned(id) {
				pinMarker = " " + pinStyle().Render("◆")
			}
			members = append(members, stateStyle(skill.State).Render(stateIcon(skill.State))+" "+displayName(skill)+pinMarker)
		}
		lines = append(lines, responsiveGrid(members, width)...)
	}

	plan := reconcile.BuildWithPins(group, m.skills, m.pins)
	active := plan.FinalActive()
	disabled := plan.FinalDisabled()
	lines = append(lines,
		"",
		helpTextStyle().Bold(true).Render("Activation Preview"),
		strings.Join([]string{
			messageStyle(messageSuccess).Render(fmt.Sprintf("Active %d", len(active))),
			messageStyle(messageError).Render(fmt.Sprintf("Disable %d", len(disabled))),
		}, "   "),
	)
	if plan.HasIssues() {
		lines = append(lines, messageStyle(messageError).Render("! Resolve issues before use"))
	}
	return strings.Join(lines, "\n")
}

func responsiveGrid(items []string, width int) []string {
	if len(items) == 0 {
		return nil
	}
	if width < 1 {
		width = 1
	}

	maxItemWidth := 1
	for _, item := range items {
		if itemWidth := ansi.StringWidth(item); itemWidth > maxItemWidth {
			maxItemWidth = itemWidth
		}
	}
	cellWidth := maxItemWidth + 2
	columns := width / cellWidth
	if columns < 1 {
		columns = 1
	}
	if columns > len(items) {
		columns = len(items)
	}
	if columns == 1 {
		return items
	}

	rows := make([]string, 0, (len(items)+columns-1)/columns)
	for start := 0; start < len(items); start += columns {
		end := start + columns
		if end > len(items) {
			end = len(items)
		}
		cells := make([]string, 0, end-start)
		for index, item := range items[start:end] {
			if index == end-start-1 {
				cells = append(cells, item)
				continue
			}
			cells = append(cells, lipgloss.NewStyle().Width(cellWidth).Render(item))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return rows
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
		pinMarker := ""
		if m.isPinned(id) {
			pinMarker = " " + pinStyle().Render("◆")
		}
		lines = append(lines, prefix+mark+" "+id+pinMarker)
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
			keyHint{key: "◆", description: "pinned"},
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
		helpLine("x", "mark or unmark a skill"),
		helpLine("b", "batch actions for marked skills"),
		helpLine("delete", "permanently remove current or marked skills"),
		helpLine("c", "clear marked skills"),
		helpLine("/", "search by ID, metadata, or group"),
		helpLine("g", "group management"),
		helpLine("p", "profile environments"),
		helpLine("n/e/d", "new, edit, or delete on management screens"),
		helpLine(":", "command palette"),
		helpLine("u", "preview and apply selected group"),
		helpLine("enter", "expand details"),
		helpLine("d", "doctor/status"),
		helpLine("◆", "pinned skill; always active"),
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
	return skillRowWithSelection(skill, selected, pinned, false)
}

func skillRowWithSelection(skill catalog.Skill, selected, pinned, marked bool) string {
	name := displayName(skill)
	if name != skill.ID {
		name = fmt.Sprintf("%s [%s]", name, skill.ID)
	}
	if selected {
		name = selectedRowStyle().Render(name)
	}
	if pinned {
		name += " " + pinStyle().Render("◆")
	}
	row := fmt.Sprintf("%s %s", stateStyle(skill.State).Render(stateIcon(skill.State)), name)
	if marked {
		row = selectionStyle().Render("✓") + " " + row
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
