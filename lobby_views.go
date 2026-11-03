package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (l lobbySnapshot) Title() string       { return l.name }
func (l lobbySnapshot) FilterValue() string { return l.name }
func (l lobbySnapshot) Description() string {
	status := "waiting"
	if l.started {
		status = "in progress"
	} else if l.players == 4 {
		status = "full"
	}
	return fmt.Sprintf("%d/4 players · %s", l.players, status)
}

type LobbyBrowserModel struct {
	list list.Model
	err  error
}

func newLobbyBrowserModel(lobbies []lobbySnapshot, width, height int) *LobbyBrowserModel {
	m := &LobbyBrowserModel{list: list.New(nil, list.NewDefaultDelegate(), width, max(1, height-2))}
	m.list.Title = "El Poblador · Lobbies"
	m.list.SetFilteringEnabled(false)
	m.list.SetShowStatusBar(false)
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "join")),
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new lobby")),
		}
	}
	m.refresh(lobbies)
	return m
}

func (m *LobbyBrowserModel) refresh(lobbies []lobbySnapshot) tea.Cmd {
	selected, hadSelection := m.list.SelectedItem().(lobbySnapshot)
	items := make([]list.Item, 0, len(lobbies))
	selectedIndex := -1
	for _, lobby := range lobbies {
		if lobby.started {
			continue
		}
		if hadSelection && lobby.id == selected.id {
			selectedIndex = len(items)
		}
		items = append(items, lobby)
	}
	cmd := m.list.SetItems(items)
	if selectedIndex >= 0 {
		m.list.Select(selectedIndex)
	}
	return cmd
}

func (m *LobbyBrowserModel) Init() tea.Cmd { return nil }

func (m *LobbyBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(1, msg.Height-2))
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			return m, func() tea.Msg { return createLobbyMsg{} }
		case "enter":
			if lobby, ok := m.list.SelectedItem().(lobbySnapshot); ok {
				return m, func() tea.Msg { return joinLobbyMsg{id: lobby.id} }
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *LobbyBrowserModel) View() string {
	view := m.list.View()
	if m.err != nil {
		view += "\n" + m.err.Error()
	}
	return view
}

type CreateLobbyModel struct {
	input textinput.Model
	err   error
}

func newCreateLobbyModel() *CreateLobbyModel {
	input := textinput.New()
	input.Placeholder = "Lobby name"
	input.CharLimit = lobbyNameLimit
	return &CreateLobbyModel{input: input}
}

func (m *CreateLobbyModel) Init() tea.Cmd { return m.input.Focus() }

func (m *CreateLobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.Width = max(1, min(lobbyNameLimit, msg.Width-4))
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return browseLobbiesMsg{} }
		case "enter":
			name := m.input.Value()
			return m, func() tea.Msg { return submitLobbyMsg{name: name} }
		default:
			m.err = nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *CreateLobbyModel) View() string {
	view := "Create a lobby\n\n" + m.input.View()
	if m.err != nil {
		view += "\n" + m.err.Error()
	}
	return view + "\n\nEnter: create · Esc: cancel · Ctrl+C: quit"
}

type lobbyRef struct {
	lobby    *lobbyCommon
	myName   string
	gameConn *GameConn
}

func (m *lobbyRef) Init() tea.Cmd { return nil }

func (m *lobbyRef) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return browseLobbiesMsg{} }
		case "enter":
			m.lobby.toggleReady(m.myName)
		}
	}
	m.lobby.m.RLock()
	defer m.lobby.m.RUnlock()
	if m.lobby.isStarted {
		playerIndex := m.lobby.players[m.myName].playerIndex
		m.gameConn.userPlayer = &playerIndex
		return m.gameConn, nil
	}
	return m, nil
}

func (m *lobbyRef) View() string {
	m.lobby.m.RLock()
	defer m.lobby.m.RUnlock()
	lines := []string{m.lobby.name, "", "Players in lobby:"}
	names := make([]string, 0, len(m.lobby.players))
	for name := range m.lobby.players {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ready := "not ready"
		if m.lobby.players[name].isReady {
			ready = "ready"
		}
		lines = append(lines, fmt.Sprintf("%s [%s]", name, ready))
	}
	lines = append(lines, "", "3–4 players required; the game starts when everyone is ready.",
		"Enter: toggle ready · Esc: leave · Ctrl+C: quit")
	return strings.Join(lines, "\n")
}
