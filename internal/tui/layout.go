package tui

import (
	"charm.land/lipgloss/v2"
	"strings"
)

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

func (m *Model) mainPanelContentHeight() int {
	height := m.height - 12
	if height < 1 {
		return 1
	}
	return height
}

func viewportBounds(total, selected, capacity int) (start, end int) {
	if total == 0 {
		return 0, 0
	}
	if capacity < 1 {
		capacity = 1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}
	if capacity >= total {
		return 0, total
	}

	start = selected - capacity + 1
	if start < 0 {
		start = 0
	}
	end = start + capacity
	if end > total {
		end = total
		start = end - capacity
	}
	return start, end
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
	if height < 3 {
		height = 3
	}
	contentWidth := column.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	border := lipgloss.NormalBorder()
	borderColor := lipgloss.Color("8")
	if column.focused {
		border = lipgloss.RoundedBorder()
		borderColor = lipgloss.Color("6")
	}
	inner := lipgloss.NewStyle().
		Width(contentWidth).
		Height(height - 2).
		MaxHeight(height - 2).
		Render(column.content)
	return lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		Render(inner)
}

func mainColumnHeader(title string, width int, focused bool) string {
	style := lipgloss.NewStyle().Width(width).PaddingLeft(1).Bold(true).Foreground(lipgloss.Color("8"))
	if focused {
		style = style.Foreground(lipgloss.Color("6")).Underline(true)
	}
	return style.Render(title)
}

func (m *Model) panel(title, content string, width, height int, focused bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	style := lipgloss.NewStyle().Width(width-2).Height(height-2).MaxHeight(height).Padding(0, 1)
	if focused {
		style = style.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("6"))
	} else {
		style = style.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8"))
	}
	return style.Render(lipgloss.NewStyle().Bold(true).Render(title) + "\n" + content)
}
