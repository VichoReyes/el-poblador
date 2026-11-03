package main

import (
	"log"
	"time"

	"git.sr.ht/~vicho/el_poblador/game"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
)

const tickPeriod = 200 * time.Millisecond

func dummyHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	conn := &GameConn{width: pty.Window.Width, height: pty.Window.Height, game: &game.Game{}}
	conn.game.Start([]string{"a", "b", "c"})
	return conn, []tea.ProgramOption{tea.WithAltScreen()}
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(tickPeriod, func(t time.Time) tea.Msg { return TickMsg(t) })
}

// Keep cleanup around the whole Bubble Tea program, including SSH disconnects.
func multiplayerMiddleware(registry *lobbyRegistry) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			pty, _, _ := s.Pty()
			model := newSessionModel(registry, s.User(), pty.Window.Width, pty.Window.Height)
			defer model.Close()
			bubbletea.Middleware(func(ssh.Session) (tea.Model, []tea.ProgramOption) {
				return model, []tea.ProgramOption{tea.WithAltScreen()}
			})(next)(s)
		}
	}
}

type browseLobbiesMsg struct{}
type createLobbyMsg struct{}
type joinLobbyMsg struct{ id uint64 }
type submitLobbyMsg struct{ name string }

// SessionModel owns navigation, terminal dimensions, and the single refresh loop.
// Only this SSH session owns its child models; the registry shares game state.
// Update and View run on Bubble Tea's event loop. Close runs there on Ctrl+C,
// or after the program stops; it must not run concurrently with that loop.
type SessionModel struct {
	registry      *lobbyRegistry
	username      string
	width, height int
	active        tea.Model
	lobby         *lobbyCommon
	closed        bool
}

func newSessionModel(registry *lobbyRegistry, username string, width, height int) *SessionModel {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 32
	}
	return &SessionModel{
		registry: registry, username: username, width: width, height: height,
		active: newLobbyBrowserModel(registry.List(), width, height),
	}
}

func (m *SessionModel) Init() tea.Cmd {
	return tea.Batch(doTick(), m.active.Init())
}

func (m *SessionModel) Close() {
	m.closed = true
	m.leaveLobby()
}

func (m *SessionModel) leaveLobby() {
	if m.lobby != nil {
		m.registry.Leave(m.lobby, m.username)
		m.lobby = nil
	}
}

func (m *SessionModel) activate(model tea.Model) tea.Cmd {
	m.active = model
	_, resize := model.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return tea.Batch(model.Init(), resize)
}

func (m *SessionModel) enterLobby(lobby *lobbyCommon) tea.Cmd {
	m.lobby = lobby
	log.Printf("player joined lobby: lobby=%q id=%d player=%q", lobby.name, lobby.id, m.username)
	return m.activate(&lobbyRef{
		lobby: lobby, myName: m.username,
		gameConn: &GameConn{game: &lobby.game},
	})
}

func (m *SessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.closed {
		return m, tea.Quit
	}
	var refresh tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.Close()
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if msg.Width <= 0 || msg.Height <= 0 {
			return m, nil
		}
		m.width, m.height = msg.Width, msg.Height
	case TickMsg:
		refresh = doTick()
		if browser, ok := m.active.(*LobbyBrowserModel); ok {
			refresh = tea.Batch(refresh, browser.refresh(m.registry.List()))
		}
	case createLobbyMsg:
		if _, ok := m.active.(*LobbyBrowserModel); ok {
			return m, m.activate(newCreateLobbyModel())
		}
		return m, nil
	case submitLobbyMsg:
		form, ok := m.active.(*CreateLobbyModel)
		if !ok || m.lobby != nil {
			return m, nil
		}
		lobby, err := m.registry.Create(msg.name, m.username)
		if err != nil {
			form.err = err
			return m, nil
		}
		return m, m.enterLobby(lobby)
	case joinLobbyMsg:
		browser, ok := m.active.(*LobbyBrowserModel)
		if !ok || m.lobby != nil {
			return m, nil
		}
		lobby, err := m.registry.Join(msg.id, m.username)
		if err != nil {
			browser.err = err
			return m, browser.refresh(m.registry.List())
		}
		return m, m.enterLobby(lobby)
	case browseLobbiesMsg:
		// Escape leaves waiting rooms; active games retain their Escape action.
		switch m.active.(type) {
		case *CreateLobbyModel, *lobbyRef:
			m.leaveLobby()
			return m, m.activate(newLobbyBrowserModel(m.registry.List(), m.width, m.height))
		}
		return m, nil
	}
	next, cmd := m.active.Update(msg)
	if next != m.active {
		cmd = tea.Batch(cmd, m.activate(next))
	}
	return m, tea.Batch(refresh, cmd)
}

func (m *SessionModel) View() string { return m.active.View() }
