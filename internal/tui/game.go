package tui

import (
	"github.com/VichoReyes/el-poblador/game"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GameConn is the Bubble tea model for an active game
type GameConn struct {
	renderer        *lipgloss.Renderer
	game            *game.Game
	width           int // TODO: abstract to ResizableModel
	height          int
	userPlayer      *int // nil means follow turn
	twoColumnCycle  int  // 0-1: for width 90-119
	oneColumnCycle  int  // 0-2: for width <90
	useEmoji        bool
	saveNoticeShown bool
}

func (m *GameConn) Init() tea.Cmd {
	return nil
}

func (m *GameConn) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.twoColumnCycle = (m.twoColumnCycle + 1) % 2
			m.oneColumnCycle = (m.oneColumnCycle + 1) % 3
		case "up", "down", "left", "right":
			m.game.MoveCursor(msg.String(), m.userPlayer)
		case "enter":
			m.game.ConfirmAction(m.userPlayer)
			if m.game.ShouldQuit() && m.game.SaveNotice() == "" {
				return m, tea.Quit
			}
		case "esc":
			m.game.CancelAction(m.userPlayer)

		case "e":
			m.useEmoji = !m.useEmoji
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	}
	if notice := m.game.SaveNotice(); notice != "" && !m.saveNoticeShown {
		m.saveNoticeShown = true
		// Print outside the managed view so the code survives quitting and remains
		// in terminal scrollback. Println must run after leaving the alternate screen.
		return m, tea.Sequence(tea.ExitAltScreen, tea.Println(notice))
	}
	return m, nil
}

func (m *GameConn) View() string {
	if notice := m.game.SaveNotice(); notice != "" {
		return ""
	}
	return m.game.Print(game.PrintOptions{
		Renderer:       m.renderer,
		Width:          m.width,
		Height:         m.height,
		RequestPlayer:  m.userPlayer,
		TwoColumnCycle: m.twoColumnCycle,
		OneColumnCycle: m.oneColumnCycle,
		UseEmoji:       m.useEmoji,
	})
}

// LocalConn is a GameConn with a selectable userPlayer
type LocalConn GameConn

func (m *LocalConn) Init() tea.Cmd {
	gc := (*GameConn)(m)
	return gc.Init()
}

func (m *LocalConn) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1", "2", "3", "4":
			player := int(msg.String()[0] - '1')
			m.userPlayer = &player
			return m, nil
		case "0":
			m.userPlayer = nil
			return m, nil
		}
	}
	gc := (*GameConn)(m)
	gc2, cmd := gc.Update(msg)
	asGameConn := gc2.(*GameConn)
	return (*LocalConn)(asGameConn), cmd
}

func (m *LocalConn) View() string {
	gc := (*GameConn)(m)
	return gc.View()
}

// NewLocalConn creates a local game model.
func NewLocalConn(g *game.Game, useEmoji bool) *LocalConn {
	return &LocalConn{game: g, useEmoji: useEmoji}
}
