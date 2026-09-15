package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/profile"
	"github.com/Kklyee/skiller/internal/transaction"
	"strings"
)

type modal uint8

const (
	modalNone modal = iota
	modalReconcile
	modalDeleteGroup
	modalGroupName
	modalDeleteProfile
	modalProfileName
	modalBatch
	modalBatchGroup
	modalPalette
)

type paletteCommand struct {
	id          string
	title       string
	description string
}

type batchAction uint8

const (
	batchActionNone batchAction = iota
	batchActionAddGroup
	batchActionRemoveGroup
)

func (m *Model) updateModal(message bubbletea.KeyPressMsg) bubbletea.Cmd {
	key := message.String()
	if m.modal == modalPalette {
		return m.updatePalette(message, key)
	}
	if m.modal == modalBatch {
		switch key {
		case "esc":
			m.modal = modalNone
			m.clearMessage()
		case "e":
			m.applyBatchVisibility(true)
		case "d":
			m.applyBatchVisibility(false)
		case "a":
			m.openBatchGroup(batchActionAddGroup)
		case "r":
			m.openBatchGroup(batchActionRemoveGroup)
		}
		return nil
	}

	if m.modal == modalBatchGroup {
		switch key {
		case "esc":
			m.modal = modalBatch
			m.batchAction = batchActionNone
		case "up", "k":
			m.moveBatchGroup(-1)
		case "down", "j":
			m.moveBatchGroup(1)
		case "enter":
			m.applyBatchGroup()
		}
		return nil
	}

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
				m.setMessage(messageSuccess, fmt.Sprintf("Applied %s %s", m.planKind, m.plan.Group))
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

	if m.modal == modalProfileName {
		switch message.String() {
		case "esc":
			m.modal = modalNone
			m.clearMessage()
		case "enter":
			name := strings.TrimSpace(m.input)
			if name == "" {
				m.setMessage(messageInfo, "Profile name is required")
				return nil
			}
			if _, err := profile.New(m.paths.Profiles).Create(name); err != nil {
				m.setError(err)
				return nil
			} else if err := m.refresh(); err != nil {
				m.setError(err)
				return nil
			} else {
				m.selectedProfile = name
				m.modal = modalNone
				m.setMessage(messageSuccess, fmt.Sprintf("Created profile %s; configure fields", name))
				m.openProfileEditor()
			}
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

	if m.modal == modalDeleteProfile {
		if key == "esc" || key == "n" {
			m.modal = modalNone
			m.clearMessage()
			return nil
		}
		if key == "enter" || strings.EqualFold(key, "y") {
			if err := profile.New(m.paths.Profiles).Delete(m.deleteProfile); err != nil {
				m.setError(err)
			} else if err := m.refresh(); err != nil {
				m.setError(err)
			} else {
				m.setMessage(messageSuccess, fmt.Sprintf("Deleted profile %s", m.deleteProfile))
			}
			m.modal = modalNone
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
			m.setMessage(messageSuccess, fmt.Sprintf("Deleted group %s", m.deleteGroup))
		}
		m.modal = modalNone
	}
	return nil
}

func (m *Model) updatePalette(message bubbletea.KeyPressMsg, key string) bubbletea.Cmd {
	switch key {
	case "esc":
		m.closePalette()
	case "up", "k":
		m.movePalette(-1)
	case "down", "j":
		m.movePalette(1)
	case "backspace", "delete":
		runes := []rune(m.paletteQuery)
		if len(runes) > 0 {
			m.paletteQuery = string(runes[:len(runes)-1])
			m.paletteIndex = 0
		}
	case "enter":
		m.executePaletteCommand()
	default:
		if message.Text != "" {
			m.paletteQuery += message.Text
			m.paletteIndex = 0
		}
	}
	return nil
}

func (m *Model) openCommandPalette() {
	m.modal = modalPalette
	m.paletteQuery = ""
	m.paletteIndex = 0
}

func (m *Model) closePalette() {
	m.modal = modalNone
	m.paletteQuery = ""
	m.paletteIndex = 0
}

func (m *Model) paletteCommands() []paletteCommand {
	return []paletteCommand{
		{id: "groups", title: "Groups", description: "open group management"},
		{id: "profiles", title: "Profiles", description: "open profile environments"},
		{id: "project", title: "Project", description: "open project environment"},
		{id: "use", title: "Use selected group", description: "preview the current group"},
		{id: "doctor", title: "Doctor", description: "inspect environment health"},
		{id: "help", title: "Help", description: "show keyboard help"},
		{id: "search", title: "Search skills", description: "filter the skill list"},
		{id: "toggle-all", title: "Toggle all visible skills", description: "activate or disable visible skills"},
	}
}

func (m *Model) filteredPaletteCommands() []paletteCommand {
	query := strings.ToLower(strings.TrimSpace(m.paletteQuery))
	commands := m.paletteCommands()
	if query == "" {
		return commands
	}
	filtered := make([]paletteCommand, 0, len(commands))
	for _, command := range commands {
		if strings.Contains(strings.ToLower(command.title), query) || strings.Contains(strings.ToLower(command.description), query) {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

func (m *Model) movePalette(delta int) {
	commands := m.filteredPaletteCommands()
	if len(commands) == 0 {
		m.paletteIndex = 0
		return
	}
	m.paletteIndex += delta
	if m.paletteIndex < 0 {
		m.paletteIndex = len(commands) - 1
	}
	if m.paletteIndex >= len(commands) {
		m.paletteIndex = 0
	}
}

func (m *Model) executePaletteCommand() {
	commands := m.filteredPaletteCommands()
	if len(commands) == 0 {
		return
	}
	if m.paletteIndex >= len(commands) {
		m.paletteIndex = len(commands) - 1
	}
	command := commands[m.paletteIndex]
	m.closePalette()
	switch command.id {
	case "groups":
		m.openGroups()
	case "profiles":
		m.openProfiles()
	case "project":
		m.openProject()
	case "use":
		m.openReconcile()
	case "doctor":
		m.doctorReport = doctor.Inspect(m.paths)
		m.screen = ScreenDoctor
	case "help":
		m.screen = ScreenHelp
	case "search":
		m.searchActive = true
		m.search = ""
	case "toggle-all":
		m.toggleAllSkills()
	}
}

func (m *Model) viewBatchModal() string {
	lines := []string{selectionStyle().Render(fmt.Sprintf("Selected %d skills", m.selectedSkillCount())), ""}
	for _, id := range m.selectedSkillIDs() {
		lines = append(lines, "  "+id)
	}
	lines = append(lines,
		"",
		helpTextStyle().Render("e  enable   d  disable"),
		helpTextStyle().Render("a  add to group   r  remove from group"),
	)
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Batch Actions"),
		m.panel("Marked Skills", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "e", description: "enable"},
			keyHint{key: "d", description: "disable"},
			keyHint{key: "a", description: "add group"},
			keyHint{key: "r", description: "remove group"},
			keyHint{key: "esc", description: "cancel"},
		),
	}, "\n")
}

func (m *Model) viewBatchGroupModal() string {
	action := "Add to group"
	if m.batchAction == batchActionRemoveGroup {
		action = "Remove from group"
	}
	lines := []string{helpTextStyle().Render(action), ""}
	for index, group := range m.groups {
		prefix := "  "
		if index == m.batchGroupIndex {
			prefix = selectedRowStyle().Render("›") + " "
		}
		lines = append(lines, prefix+groupNameStyle().Render(group.Name))
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Batch Group"),
		m.panel("Select Group", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "↑↓/jk", description: "move"},
			keyHint{key: "enter", description: "apply"},
			keyHint{key: "esc", description: "back"},
		),
	}, "\n")
}

func (m *Model) viewGroupNameModal() string {
	lines := []string{
		helpTextStyle().Render("Create a new group"),
		"",
		helpTextStyle().Render("Group name"),
		selectedRowStyle().Render(m.input + "▌"),
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Groups"),
		m.panel("New Group", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "type", description: "name"},
			keyHint{key: "enter", description: "create"},
			keyHint{key: "esc", description: "cancel"},
		),
	}, "\n")
}

func (m *Model) viewProfileNameModal() string {
	lines := []string{
		helpTextStyle().Render("Create a reusable skill environment"),
		"",
		helpTextStyle().Render("Profile name"),
		selectedRowStyle().Render(m.input + "▌"),
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Profiles"),
		m.panel("New Profile", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "type", description: "name"},
			keyHint{key: "enter", description: "create"},
			keyHint{key: "esc", description: "cancel"},
		),
	}, "\n")
}

func (m *Model) viewDeleteProfileModal() string {
	lines := []string{
		messageStyle(messageError).Render("Delete profile " + m.deleteProfile + "?"),
		"",
		helpTextStyle().Render("This removes profile metadata only; skills and groups are not changed."),
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Profiles"),
		m.panel("Confirm Delete", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "enter/y", description: "delete"},
			keyHint{key: "esc/n", description: "cancel"},
		),
	}, "\n")
}

func (m *Model) viewDeleteGroupModal() string {
	lines := []string{
		messageStyle(messageError).Render("Delete group " + m.deleteGroup + "?"),
		"",
		helpTextStyle().Render("This removes group metadata only; installed skills are not changed."),
	}
	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Groups"),
		m.panel("Confirm Delete", strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(
			keyHint{key: "enter/y", description: "delete"},
			keyHint{key: "esc/n", description: "cancel"},
		),
	}, "\n")
}

func (m *Model) viewReconcileModal() string {
	if m.planKind == "group" {
		return m.viewActivationModal()
	}
	return m.viewReconcileDiffModal()
}

func (m *Model) viewActivationModal() string {
	active := m.plan.FinalActive()
	disabled := m.plan.FinalDisabled()
	lines := []string{reconcileSectionStyle(messageSuccess, "Active", len(active))}
	for _, id := range active {
		lines = append(lines, messageStyle(messageSuccess).Render("  ● "+id))
	}
	lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageError, "Disable", len(disabled)))
	for _, id := range disabled {
		lines = append(lines, messageStyle(messageError).Render("  ○ "+id))
	}
	if len(m.plan.Missing) > 0 {
		lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageInfo, "Missing", len(m.plan.Missing)))
		for _, id := range m.plan.Missing {
			lines = append(lines, messageStyle(messageInfo).Render("  ? "+id))
		}
	}
	if len(m.plan.Issues) > 0 {
		lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageError, "Issues", len(m.plan.Issues)))
		for _, issue := range m.plan.Issues {
			lines = append(lines, messageStyle(messageError).Render("  ! "+issue))
		}
	}
	lines = append(lines,
		m.reconcileDivider(),
		"",
		helpTextStyle().Bold(true).Render("Final state"),
		strings.Join([]string{
			messageStyle(messageSuccess).Render(fmt.Sprintf("%d active", len(active))),
			messageStyle(messageError).Render(fmt.Sprintf("%d disable", len(disabled))),
		}, "  "),
	)
	if m.plan.HasIssues() {
		lines = append(lines, "Cannot apply until issues are resolved")
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}

	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Reconcile"),
		m.panel(m.reconcileTitle(), strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(keyHint{key: "enter", description: "apply"}, keyHint{key: "esc", description: "cancel"}),
	}, "\n")
}

func (m *Model) viewReconcileDiffModal() string {
	lines := []string{reconcileSectionStyle(messageSuccess, "Enable", len(m.plan.Enable))}
	for _, id := range m.plan.Enable {
		lines = append(lines, messageStyle(messageSuccess).Render("  + "+id))
	}
	lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageError, "Disable", len(m.plan.Disable)))
	for _, id := range m.plan.Disable {
		lines = append(lines, messageStyle(messageError).Render("  - "+id))
	}
	lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageInfo, "Keep", len(m.plan.Keep)))
	if len(m.plan.KeepActive)+len(m.plan.KeepDisabled) == len(m.plan.Keep) {
		if len(m.plan.KeepActive) > 0 {
			lines = append(lines, messageStyle(messageSuccess).Render(fmt.Sprintf("  active (%d)", len(m.plan.KeepActive))))
			for _, id := range m.plan.KeepActive {
				lines = append(lines, messageStyle(messageSuccess).Render("    = "+id))
			}
		}
		if len(m.plan.KeepDisabled) > 0 {
			lines = append(lines, helpTextStyle().Render(fmt.Sprintf("  disabled (%d)", len(m.plan.KeepDisabled))))
			for _, id := range m.plan.KeepDisabled {
				lines = append(lines, helpTextStyle().Render("    = "+id))
			}
		}
	} else {
		for _, id := range m.plan.Keep {
			lines = append(lines, helpTextStyle().Render("  = "+id))
		}
	}
	if len(m.plan.Missing) > 0 {
		lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageInfo, "Missing", len(m.plan.Missing)))
		for _, id := range m.plan.Missing {
			lines = append(lines, messageStyle(messageInfo).Render("  ? "+id))
		}
	}
	if len(m.plan.Issues) > 0 {
		lines = append(lines, m.reconcileDivider(), reconcileSectionStyle(messageError, "Issues", len(m.plan.Issues)))
		for _, issue := range m.plan.Issues {
			lines = append(lines, messageStyle(messageError).Render("  ! "+issue))
		}
	}
	lines = append(lines,
		m.reconcileDivider(),
		"",
		helpTextStyle().Bold(true).Render("Summary"),
		strings.Join([]string{
			messageStyle(messageSuccess).Render(fmt.Sprintf("%d enable", len(m.plan.Enable))),
			messageStyle(messageError).Render(fmt.Sprintf("%d disable", len(m.plan.Disable))),
			helpTextStyle().Render(fmt.Sprintf("%d unchanged", len(m.plan.Keep))),
		}, "  "),
	)
	if len(m.plan.KeepActive)+len(m.plan.KeepDisabled) == len(m.plan.Keep) {
		lines = append(lines, helpTextStyle().Render(fmt.Sprintf("(%d active, %d disabled)", len(m.plan.KeepActive), len(m.plan.KeepDisabled))))
	}
	if m.plan.HasIssues() {
		lines = append(lines, "Cannot apply until issues are resolved")
	}
	if m.message != "" {
		lines = append(lines, "", m.renderedMessage())
	}

	return strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Skiller / Reconcile"),
		m.panel(m.reconcileTitle(), strings.Join(lines, "\n"), m.width, m.height-3, true),
		renderKeyHints(keyHint{key: "enter", description: "apply"}, keyHint{key: "esc", description: "cancel"}),
	}, "\n")
}

func reconcileSectionStyle(kind messageKind, title string, count int) string {
	return messageStyle(kind).Render(fmt.Sprintf("%s (%d)", title, count))
}

func (m *Model) reconcileDivider() string {
	width := m.width - 8
	if width < 20 {
		width = 20
	}
	return helpTextStyle().Render(strings.Repeat("─", width))
}

func (m *Model) reconcileTitle() string {
	label := "Group"
	if m.planKind == "profile" {
		label = "Profile"
	} else if m.planKind == "project" {
		label = "Project"
	}
	return "Activate " + label + ": " + m.plan.Group
}
