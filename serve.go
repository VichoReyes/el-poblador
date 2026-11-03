package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~vicho/el_poblador/game"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/ssh"
)

const tickPeriod = 200 * time.Millisecond

func dummyHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	gameConn := &GameConn{
		width:  pty.Window.Width,
		height: pty.Window.Height,
		game:   &game.Game{},
	}
	dummyNames := []string{"a", "b", "c"}
	gameConn.game.Start(dummyNames)
	return gameConn, []tea.ProgramOption{tea.WithAltScreen()}
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(tickPeriod, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// UpdatingConn is a GameConn that updates every 0.2s
type UpdatingConn GameConn

func (m *UpdatingConn) Init() tea.Cmd {
	tickMsg := doTick()
	gc := (*GameConn)(m)
	existing := gc.Init()
	return tea.Batch(tickMsg, existing)
}

func (m *UpdatingConn) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		return m, doTick()
	}
	gc := (*GameConn)(m)
	gc2, cmd := gc.Update(msg)
	asGameConn := gc2.(*GameConn)
	return (*UpdatingConn)(asGameConn), cmd

}

func (m *UpdatingConn) View() string {
	gc := (*GameConn)(m)
	return gc.View()
}

type lobbyPlayer struct {
	isReady     bool
	playerIndex int
}

type lobbyCommon struct {
	m         sync.RWMutex
	name      string
	players   map[string]*lobbyPlayer
	isStarted bool
	game      game.Game
	start     sync.Once
	boardInit sync.Once
}

type lobbyRef struct {
	lobby    *lobbyCommon
	myName   string
	gameConn *UpdatingConn
}

var singleLobby = &lobbyCommon{name: "Game Lobby"}

func singleGameHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	lobby := singleLobby
	err := lobby.Join(s.User())
	if err != nil {
		// TODO test, what does this do?
		return nil, nil
	}
	lobby.boardInit.Do(func() { lobby.game.MakeBoard() })
	lr := &lobbyRef{
		lobby:  lobby,
		myName: s.User(),
		gameConn: &UpdatingConn{
			// TODO: width and height (but keep up to date)
			game: &lobby.game,
		},
	}

	return lr, []tea.ProgramOption{tea.WithAltScreen()}
}

func (lc *lobbyCommon) Name() string {
	return lc.name
}

func (lr *lobbyRef) Init() tea.Cmd {
	return doTick() // redraw every 1 sec
}

func (lr *lobbyRef) View() string {
	lr.lobby.m.RLock()
	defer lr.lobby.m.RUnlock()
	lines := []string{
		"Players in lobby:",
	}
	for name, p := range lr.lobby.players {
		var readiness string
		if p.isReady {
			readiness = "ready"
		} else {
			readiness = "not ready"
		}
		lines = append(lines, fmt.Sprintf("%v [%v]", name, readiness))
	}
	// TODO: show rendered board
	return strings.Join(lines, "\n")
}

func (lr *lobbyRef) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		return lr, doTick()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return nil, tea.Quit
		case "enter":
			lr.lobby.m.Lock()
			// there should be a "+= 1 but for bools"
			lr.lobby.players[lr.myName].isReady =
				!lr.lobby.players[lr.myName].isReady

			if lr.lobby.allReady() {
				lr.lobby.start.Do(func() {
					playerNames := []string{}
					// this is basically the shuffle
					i := 0
					for name, p := range lr.lobby.players {
						playerNames = append(playerNames, name)
						p.playerIndex = i
						i++
					}
					lr.lobby.game.StartOptions(playerNames, false, false)
					lr.lobby.isStarted = true
				})
			}
			lr.lobby.m.Unlock()
		}
		/* this would be a dirty hack
		case tea.WindowSizeMsg:
			// make game react to window changes while in lobby?
			lr.gameConn.Update(msg)
		*/
	}
	// lots of mutex contention here potentially
	lr.lobby.m.RLock()
	defer lr.lobby.m.RUnlock()
	if lr.lobby.isStarted {
		playerIndex := lr.lobby.players[lr.myName].playerIndex
		lr.gameConn.userPlayer = &playerIndex
		return lr.gameConn, nil
	}
	return lr, nil
}

// locks mutex
func (lc *lobbyCommon) Join(username string) error {
	lc.m.Lock()
	defer lc.m.Unlock()
	if lc.players == nil {
		lc.players = make(map[string]*lobbyPlayer)
	}
	if len(lc.players) >= 4 {
		return errors.New("lobby full")
	}
	if lc.players[username] != nil {
		return errors.New("username already taken")
	}
	lc.players[username] = &lobbyPlayer{}
	return nil
}

// allReady returns true if 3-4 players and all ready
// does not lock mutex
func (lc *lobbyCommon) allReady() bool {
	if len(lc.players) < 3 {
		return false
	}
	for _, p := range lc.players {
		if !p.isReady {
			return false
		}
	}
	return true
}
