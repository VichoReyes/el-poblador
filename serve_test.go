package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Execute only the immediate navigation intent; refresh and cursor timers are
// delivered explicitly by tests rather than starting a terminal event loop.
func navigate(t *testing.T, session *SessionModel, key tea.KeyType, runes ...rune) {
	t.Helper()
	_, cmd := session.Update(tea.KeyMsg{Type: key, Runes: runes})
	if cmd == nil {
		t.Fatal("expected a navigation command")
	}
	msg := cmd()
	switch msg.(type) {
	case createLobbyMsg, submitLobbyMsg, joinLobbyMsg, browseLobbiesMsg:
	default:
		t.Fatalf("expected a navigation intent, got %T", msg)
	}
	session.Update(msg)
}

func createTestLobby(t *testing.T, session *SessionModel, name string) {
	t.Helper()
	navigate(t, session, tea.KeyRunes, 'n')
	session.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)})
	navigate(t, session, tea.KeyEnter)
	if _, ok := session.active.(*lobbyRef); !ok {
		t.Fatalf("expected waiting room, got %T: %s", session.active, session.View())
	}
}

func TestLobbySessionsKeepPlayerNames(t *testing.T) {
	for _, playerCount := range []int{3, 4} {
		t.Run(fmt.Sprintf("%d_players", playerCount), func(t *testing.T) {
			// Exercise map iteration orders without assuming turn order matches
			// either connection or readiness order.
			for attempt := 0; attempt < 20; attempt++ {
				registry := newLobbyRegistry()
				sessions := make([]*SessionModel, playerCount)
				for i := range sessions {
					sessions[i] = newSessionModel(registry, fmt.Sprintf("name%d", i+1), 80, 32)
					if i == 0 {
						createTestLobby(t, sessions[i], "Test lobby")
					} else {
						navigate(t, sessions[i], tea.KeyEnter)
					}
					sessions[i].Update(tea.WindowSizeMsg{Width: 120 + i, Height: 40 + i})
				}
				for i := len(sessions) - 1; i >= 0; i-- {
					sessions[i].Update(tea.KeyMsg{Type: tea.KeyEnter})
				}
				for i, session := range sessions {
					session.Update(TickMsg{})
					conn, ok := session.active.(*GameConn)
					if !ok {
						t.Fatalf("%s did not enter the game on a tick (model %T)", session.username, session.active)
					}
					if conn.userPlayer == nil {
						t.Fatalf("%s has no player assignment", session.username)
					}
					index := *conn.userPlayer
					if index < 0 || index >= len(conn.game.Players) {
						t.Fatalf("%s has invalid player index %d", session.username, index)
					}
					if got := conn.game.Players[index].Name; got != session.username {
						t.Errorf("attempt %d: session %s became player %s", attempt, session.username, got)
					}
					if conn.width != 120+i || conn.height != 40+i {
						t.Errorf("%s lost its terminal dimensions: %dx%d", session.username, conn.width, conn.height)
					}
				}
			}
		})
	}
}

func TestSeparateLobbiesAndCleanup(t *testing.T) {
	registry := newLobbyRegistry()
	first := newSessionModel(registry, "alice", 80, 32)
	second := newSessionModel(registry, "alice", 80, 32)
	createTestLobby(t, first, "First")
	createTestLobby(t, second, "Second")
	if first.lobby == second.lobby || &first.lobby.game == &second.lobby.game || first.lobby.game.Board == second.lobby.game.Board {
		t.Fatal("separate lobbies share game state")
	}

	for _, name := range []string{"bob", "carol"} {
		guest := newSessionModel(registry, name, 80, 32)
		navigate(t, guest, tea.KeyEnter)
		guest.Update(tea.KeyMsg{Type: tea.KeyEnter})
		t.Cleanup(guest.Close)
	}
	first.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if _, ok := first.active.(*GameConn); !ok {
		t.Fatal("first lobby did not start")
	}
	if second.lobby.isStarted || len(second.lobby.players) != 1 {
		t.Fatal("starting the first lobby affected the second")
	}
	if _, err := registry.Join(first.lobby.id, "latecomer"); err == nil {
		t.Fatal("joined a game after it started")
	}

	navigate(t, second, tea.KeyEsc)
	if second.lobby != nil || len(registry.List()) != 1 {
		t.Fatal("leaving the last seat did not remove the lobby")
	}
	createTestLobby(t, second, "First again")
	second.Close() // The SSH middleware uses this same cleanup on disconnect.
	second.Close()
	if len(registry.List()) != 1 {
		t.Fatal("disconnect cleanup did not remove the empty lobby")
	}
	first.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if first.lobby != nil {
		t.Fatal("quit retained lobby membership")
	}
}

func TestBrowserRefreshPreservesSelectionAndReportsStaleLobby(t *testing.T) {
	registry := newLobbyRegistry()
	selected, _ := registry.Create("Bravo", "host")
	session := newSessionModel(registry, "guest", 80, 32)
	registry.Create("Alpha", "other")
	session.Update(TickMsg{})
	browser := session.active.(*LobbyBrowserModel)
	if got := browser.list.SelectedItem().(lobbySnapshot).id; got != selected.id {
		t.Fatal("refresh changed selection when an earlier row was inserted")
	}
	registry.Leave(selected, "host")
	navigate(t, session, tea.KeyEnter)
	if session.lobby != nil || browser.err == nil {
		t.Fatal("stale selection did not produce an error in the browser")
	}
	if len(browser.list.Items()) != 1 {
		t.Fatal("browser did not refresh after failed join")
	}
}

func TestBrowserHidesStartedGamesOnEntryAndRefresh(t *testing.T) {
	registry := newLobbyRegistry()
	starting := makeTestLobby(t, registry, "Alpha", "alice", "bob", "carol")
	waiting := makeTestLobby(t, registry, "Bravo", "host")
	session := newSessionModel(registry, "observer", 80, 32)
	session.Update(tea.KeyMsg{Type: tea.KeyDown}) // Keep Bravo selected as Alpha starts.
	for _, name := range []string{"alice", "bob", "carol"} {
		starting.toggleReady(name)
	}
	session.Update(TickMsg{})
	fresh := newSessionModel(registry, "new observer", 80, 32)
	for _, browser := range []*SessionModel{session, fresh} {
		items := browser.active.(*LobbyBrowserModel).list.Items()
		if len(items) != 1 || items[0].(lobbySnapshot).id != waiting.id {
			t.Fatalf("browser did not filter the started game: %+v", items)
		}
		if selected := browser.active.(*LobbyBrowserModel).list.SelectedItem().(lobbySnapshot); selected.id != waiting.id {
			t.Fatalf("filtering lost the waiting lobby selection: %+v", selected)
		}
	}
	navigate(t, session, tea.KeyEnter)
	if session.lobby != waiting {
		t.Fatal("filtered selection joined the wrong lobby")
	}
}

func TestBrowserJoiningJustStartedGameShowsErrorAndRefreshes(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Starting", "alice", "bob", "carol")
	session := newSessionModel(registry, "latecomer", 80, 32)
	for _, name := range []string{"alice", "bob", "carol"} {
		lobby.toggleReady(name)
	}
	// Join the stale selection before the browser gets its next refresh tick.
	navigate(t, session, tea.KeyEnter)
	browser := session.active.(*LobbyBrowserModel)
	if session.lobby != nil || browser.err == nil || len(browser.list.Items()) != 0 {
		t.Fatal("stale join did not reject and hide the started game")
	}
	// Enter on the now-empty browser must not attempt another join.
	session.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if session.lobby != nil {
		t.Fatal("empty browser joined a game")
	}
}

func TestLobbyNameFormValidationAndCancel(t *testing.T) {
	registry := newLobbyRegistry()
	session := newSessionModel(registry, "alice", 80, 32)
	navigate(t, session, tea.KeyRunes, 'n')
	form := session.active.(*CreateLobbyModel)
	if !form.input.Focused() {
		t.Fatal("name input was not focused on entry")
	}
	navigate(t, session, tea.KeyEnter)
	if form.err == nil || len(registry.List()) != 0 {
		t.Fatal("empty name was accepted")
	}
	navigate(t, session, tea.KeyEsc)
	if _, ok := session.active.(*LobbyBrowserModel); !ok {
		t.Fatal("cancel did not return to the browser")
	}
	createTestLobby(t, session, "  Ñuñoa  ")
	if session.lobby.name != "Ñuñoa" {
		t.Fatal("lobby name was not trimmed or Unicode was lost")
	}
	for _, name := range []string{"ñuñoa", strings.Repeat("a", lobbyNameLimit+1), "bad\x1bname"} {
		if _, err := registry.Create(name, "other"); err == nil {
			t.Errorf("accepted invalid or duplicate lobby name %q", name)
		}
	}
	other := newSessionModel(registry, "other", 80, 32)
	navigate(t, other, tea.KeyRunes, 'n')
	other.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ñuñoa")})
	navigate(t, other, tea.KeyEnter)
	if form := other.active.(*CreateLobbyModel); form.err == nil {
		t.Fatal("duplicate name error was not shown in the form")
	}
}

func TestSessionIgnoresUnknownTerminalSize(t *testing.T) {
	session := newSessionModel(newLobbyRegistry(), "alice", 0, 0)
	session.Update(tea.WindowSizeMsg{})
	if session.width != 80 || session.height != 32 || !strings.Contains(session.View(), "Lobbies") {
		t.Fatal("unknown SSH terminal size replaced the usable fallback dimensions")
	}
}

func TestConcurrentJoinsRespectCapacity(t *testing.T) {
	registry := newLobbyRegistry()
	lobby, _ := registry.Create("Capacity", "host")
	registry.Join(lobby.id, "second")
	registry.Join(lobby.id, "third")
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := registry.Join(lobby.id, fmt.Sprintf("guest%d", i))
			results <- err
			registry.List()
		}(i)
	}
	wg.Wait()
	close(results)
	joined := 0
	for err := range results {
		if err == nil {
			joined++
		}
	}
	if joined != 1 || registry.List()[0].players != 4 {
		t.Fatalf("expected one remaining seat to be filled, got %d successful joins", joined)
	}
	if _, err := registry.Join(lobby.id, "fifth"); err == nil {
		t.Fatal("joined a full lobby")
	}
}

func TestLeavingUnreadySeatStartsReadyPlayers(t *testing.T) {
	registry := newLobbyRegistry()
	sessions := make([]*SessionModel, 4)
	for i := range sessions {
		sessions[i] = newSessionModel(registry, fmt.Sprintf("name%d", i), 80, 32)
		if i == 0 {
			createTestLobby(t, sessions[i], "Ready players")
		} else {
			navigate(t, sessions[i], tea.KeyEnter)
		}
	}
	for _, session := range sessions[:3] {
		session.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	if sessions[0].lobby.isStarted {
		t.Fatal("game started while the fourth player was not ready")
	}
	navigate(t, sessions[3], tea.KeyEsc)
	for _, session := range sessions[:3] {
		session.Update(TickMsg{})
		if _, ok := session.active.(*GameConn); !ok {
			t.Fatal("remaining ready players did not enter the game")
		}
	}
}

func TestConcurrentSessionsStartTogether(t *testing.T) {
	registry := newLobbyRegistry()
	sessions := make([]*SessionModel, 4)
	for i := range sessions {
		sessions[i] = newSessionModel(registry, fmt.Sprintf("name%d", i), 80, 32)
		if i == 0 {
			createTestLobby(t, sessions[i], "Concurrent start")
		} else {
			navigate(t, sessions[i], tea.KeyEnter)
		}
	}
	var wg sync.WaitGroup
	for _, session := range sessions {
		wg.Add(1)
		go func(session *SessionModel) {
			defer wg.Done()
			session.Update(tea.KeyMsg{Type: tea.KeyEnter})
			session.Update(TickMsg{})
			session.View()
			registry.List()
		}(session)
	}
	wg.Wait()
	for _, session := range sessions {
		session.Update(TickMsg{})
		conn, ok := session.active.(*GameConn)
		if !ok {
			t.Fatalf("%s did not enter the game", session.username)
		}
		if conn.game.Players[*conn.userPlayer].Name != session.username {
			t.Fatalf("%s changed identity during concurrent start", session.username)
		}
	}
}

func TestQuitBeforePendingGameTransition(t *testing.T) {
	registry := newLobbyRegistry()
	sessions := make([]*SessionModel, 3)
	for i := range sessions {
		sessions[i] = newSessionModel(registry, fmt.Sprintf("name%d", i), 80, 32)
		if i == 0 {
			createTestLobby(t, sessions[i], "Pending transition")
		} else {
			navigate(t, sessions[i], tea.KeyEnter)
		}
	}
	for _, session := range sessions {
		session.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	// The first session is still in its waiting room when it quits. A queued
	// tick must not try to access the membership that cleanup just removed.
	sessions[0].Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_, cmd := sessions[0].Update(TickMsg{})
	if cmd == nil {
		t.Fatal("closed session did not quit on a pending message")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("closed session scheduled more work")
	}
	if got := registry.List()[0].players; got != 2 {
		t.Fatalf("quit left %d connected players, expected 2", got)
	}
}
