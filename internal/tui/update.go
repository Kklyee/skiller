package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"github.com/Kklyee/skiller/internal/doctor"
)

func (m *Model) Init() bubbletea.Cmd {
	return m.resizePoll()
}

func (m *Model) Update(message bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch message := message.(type) {
	case bubbletea.WindowSizeMsg:
		return m, m.applyWindowSize(message.Width, message.Height)
	case resizePollMsg:
		pollCmd := m.resizePoll()
		if !message.valid {
			return m, pollCmd
		}
		resizeCmd := func() bubbletea.Msg {
			return bubbletea.WindowSizeMsg{Width: message.width, Height: message.height}
		}
		return m, bubbletea.Batch(resizeCmd, pollCmd)
	case startupTickMsg:
		if !m.startupActive {
			return m, nil
		}
		if m.startupFrame >= startupFrameCount-1 {
			m.finishStartup()
			return m, nil
		}
		m.startupFrame++
		return m, m.startupTick()
	case bubbletea.KeyPressMsg:
		if m.startupActive {
			m.finishStartup()
			return m, nil
		}
		return m, m.updateKey(message)
	}

	return m, nil
}

func (m *Model) applyWindowSize(width, height int) bubbletea.Cmd {
	if width <= 0 || height <= 0 {
		return nil
	}
	m.width = width
	m.height = height
	if m.startupPending {
		m.startupPending = false
		if m.width >= 60 && m.height >= 8 {
			m.startupActive = true
			m.startupFrame = 0
			return m.startupTick()
		}
	}
	if m.startupActive && (m.width < 60 || m.height < 8) {
		m.finishStartup()
	}
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
			m.openGroups()
		} else {
			m.detailExpanded = true
		}
	case "esc":
		m.detailExpanded = false
		m.focus = FocusSkills
	case "g":
		m.openGroups()
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

func (m *Model) openGroups() {
	m.screen = ScreenGroups
	m.normalizeGroupSelection()
	m.detailExpanded = false
	m.clearMessage()
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
