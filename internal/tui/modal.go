package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Kklyee/skiller/internal/group"
	"github.com/Kklyee/skiller/internal/transaction"
	"strings"
)

type modal uint8

const (
	modalNone modal = iota
	modalReconcile
	modalDeleteGroup
	modalGroupName
	modalBatch
	modalBatchGroup
)

type batchAction uint8

const (
	batchActionNone batchAction = iota
	batchActionAddGroup
	batchActionRemoveGroup
)

func (m *Model) updateModal(message bubbletea.KeyPressMsg) bubbletea.Cmd {
	key := message.String()
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
	lines := []string{messageStyle(messageSuccess).Render("Enable")}
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
	lines = append(lines,
		"",
		helpTextStyle().Bold(true).Render("Summary"),
		strings.Join([]string{
			messageStyle(messageSuccess).Render(fmt.Sprintf("%d enable", len(m.plan.Enable))),
			messageStyle(messageError).Render(fmt.Sprintf("%d disable", len(m.plan.Disable))),
			helpTextStyle().Render(fmt.Sprintf("%d unchanged", len(m.plan.Keep))),
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

func (m *Model) reconcileTitle() string {
	label := "Group"
	if m.planKind == "profile" {
		label = "Profile"
	}
	return "Activate " + label + ": " + m.plan.Group
}
