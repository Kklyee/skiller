package tui

import (
	bubbletea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Kklyee/skiller/internal/paths"
	"github.com/charmbracelet/x/term"
	"os"
	"strings"
	"time"
)

type startupTickMsg struct{}

type resizePollMsg struct {
	width  int
	height int
	valid  bool
}

const startupFrameCount = 4
const startupFrameInterval = 160 * time.Millisecond
const resizePollInterval = 250 * time.Millisecond

func Run(pathSet paths.Set) error {
	model, err := NewModel(pathSet)
	if err != nil {
		return err
	}
	if interactiveTerminal() {
		model.queueStartupAnimation()
	}

	_, err = bubbletea.NewProgram(&model).Run()
	return err
}

func (m *Model) resizePoll() bubbletea.Cmd {
	return bubbletea.Tick(resizePollInterval, func(time.Time) bubbletea.Msg {
		width, height, err := term.GetSize(os.Stdout.Fd())
		if err != nil {
			return resizePollMsg{}
		}
		return resizePollMsg{width: width, height: height, valid: true}
	})
}

func (m *Model) queueStartupAnimation() {
	if !startupAnimationAllowed() {
		return
	}
	m.startupPending = true
	m.startupActive = false
	m.startupFrame = 0
}

func startupAnimationAllowed() bool {
	return os.Getenv("NO_COLOR") == "" && !strings.EqualFold(os.Getenv("TERM"), "dumb")
}

func interactiveTerminal() bool {
	for _, file := range []*os.File{os.Stdin, os.Stdout} {
		info, err := file.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

func (m *Model) startupTick() bubbletea.Cmd {
	return bubbletea.Tick(startupFrameInterval, func(time.Time) bubbletea.Msg {
		return startupTickMsg{}
	})
}

func (m *Model) finishStartup() {
	m.startupPending = false
	m.startupActive = false
	m.startupFrame = startupFrameCount - 1
}

func (m *Model) viewStartup() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, startupLogo(m.startupFrame))
}

func startupLogo(frame int) string {
	if frame < 0 {
		frame = 0
	}
	if frame >= startupFrameCount {
		frame = startupFrameCount - 1
	}

	boxStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	secondaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	switch frame {
	case 0:
		return boxStyle.Render("·")
	case 1:
		return strings.Join([]string{
			boxStyle.Render("╭─╮"),
			boxStyle.Render("╰─╮"),
			boxStyle.Render("╰─╯"),
		}, "\n")
	case 2:
		return strings.Join([]string{
			boxStyle.Render("╭─╮"),
			boxStyle.Render("╰─╮") + "  " + brandStyle.Render("SKILLER"),
			boxStyle.Render("╰─╯"),
		}, "\n")
	default:
		return strings.Join([]string{
			boxStyle.Render("╭─╮"),
			boxStyle.Render("╰─╮") + "  " + brandStyle.Render("SKILLER"),
			boxStyle.Render("╰─╯") + "  " + secondaryStyle.Render("Skill Visibility Manager"),
		}, "\n")
	}
}
