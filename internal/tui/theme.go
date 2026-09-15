package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/Kklyee/skiller/internal/catalog"
	"github.com/Kklyee/skiller/internal/doctor"
	"github.com/Kklyee/skiller/internal/environment"
)

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

func doctorLevelStyle(level doctor.Level) lipgloss.Style {
	color := lipgloss.Color("8")
	switch level {
	case doctor.Healthy:
		color = lipgloss.Color("10")
	case doctor.Warning:
		color = lipgloss.Color("11")
	case doctor.Error:
		color = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func doctorSectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
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

func selectedRowStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
}

func groupNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
}

func pinStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
}

func selectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
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

func environmentStatusStyle(status environment.Status) lipgloss.Style {
	color := lipgloss.Color("8")
	switch status {
	case environment.StatusSynced:
		color = lipgloss.Color("10")
	case environment.StatusModified:
		color = lipgloss.Color("11")
	case environment.StatusNeedsAttention:
		color = lipgloss.Color("9")
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func environmentStatusIcon(status environment.Status) string {
	switch status {
	case environment.StatusSynced:
		return "●"
	case environment.StatusModified:
		return "◐"
	case environment.StatusNeedsAttention:
		return "!"
	default:
		return "○"
	}
}

func environmentStatusLabel(status environment.Status) string {
	switch status {
	case environment.StatusSynced:
		return "Applied"
	case environment.StatusModified:
		return "Modified"
	case environment.StatusNeedsAttention:
		return "Needs attention"
	default:
		return "Not applied"
	}
}
