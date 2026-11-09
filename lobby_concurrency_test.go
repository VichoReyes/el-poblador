package main

import (
	"fmt"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Release all workers together to exercise competing lock acquisition paths.
// A SessionModel still has only one worker, matching Bubble Tea's event loop.
func runLobbyWorkers(workers ...func()) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, worker := range workers {
		wg.Add(1)
		go func(worker func()) {
			defer wg.Done()
			<-start
			worker()
		}(worker)
	}
	close(start)
	wg.Wait()
}

func TestLobbyConcurrentCreateReservesNameOnce(t *testing.T) {
	registry := newLobbyRegistry()
	const count = 16
	results := make(chan *lobbyCommon, count)
	workers := make([]func(), count)
	for i := range workers {
		workers[i] = func() {
			name := "Same name"
			if i%2 == 0 {
				name = "  SAME NAME  "
			}
			lobby, _ := registry.Create(name, fmt.Sprintf("host%d", i))
			results <- lobby
		}
	}
	runLobbyWorkers(workers...)
	close(results)
	winners := 0
	for lobby := range results {
		if lobby != nil {
			winners++
		}
	}
	if rows := registry.List(); winners != 1 || len(rows) != 1 || rows[0].players != 1 {
		t.Fatalf("duplicate create produced %d winners and %+v", winners, rows)
	}
}

func TestLobbyConcurrentCreateUsesDistinctIDs(t *testing.T) {
	registry := newLobbyRegistry()
	const count = 16
	workers := make([]func(), count)
	for i := range workers {
		workers[i] = func() {
			if _, err := registry.Create(fmt.Sprintf("Lobby %02d", i), "host"); err != nil {
				t.Error(err)
			}
			registry.List()
		}
	}
	runLobbyWorkers(workers...)
	rows := registry.List()
	if len(rows) != count {
		t.Fatalf("got %d lobbies, want %d", len(rows), count)
	}
	ids := make(map[uint64]bool)
	for _, row := range rows {
		if ids[row.id] {
			t.Errorf("duplicate lobby ID %d", row.id)
		}
		ids[row.id] = true
	}
}

func TestLobbyConcurrentDuplicateUsernameJoinsOnce(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Duplicate joins", "host")
	const count = 12
	results := make(chan error, count)
	workers := make([]func(), count)
	for i := range workers {
		workers[i] = func() {
			_, err := registry.Join(lobby.id, "guest")
			results <- err
		}
	}
	runLobbyWorkers(workers...)
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		}
	}
	if winners != 1 || registry.List()[0].players != 2 {
		t.Fatalf("duplicate username had %d successful joins", winners)
	}
}

func TestLobbyConcurrentJoinAndLastLeave(t *testing.T) {
	for attempt := 0; attempt < 40; attempt++ {
		registry := newLobbyRegistry()
		lobby := makeTestLobby(t, registry, "Last seat", "host")
		var joined *lobbyCommon
		var joinErr error
		runLobbyWorkers(
			func() { joined, joinErr = registry.Join(lobby.id, "guest") },
			func() { registry.Leave(lobby, "host") },
			func() {
				for i := 0; i < 16; i++ {
					for _, row := range registry.List() {
						if row.players == 0 {
							t.Error("browser saw a published empty lobby")
						}
					}
				}
			},
		)
		rows := registry.List()
		if joinErr == nil {
			if joined != lobby || len(rows) != 1 || rows[0].players != 1 {
				t.Fatalf("successful join was lost to concurrent cleanup: %+v", rows)
			}
			registry.Leave(joined, "guest")
		} else if len(rows) != 0 {
			t.Fatalf("failed join left an empty lobby registered: %+v", rows)
		}
	}
}

func TestLobbyConcurrentReadyAndFourthJoin(t *testing.T) {
	for attempt := 0; attempt < 40; attempt++ {
		registry := newLobbyRegistry()
		lobby := makeTestLobby(t, registry, "Fourth seat", "alice", "bob", "carol")
		lobby.toggleReady("alice")
		lobby.toggleReady("bob")
		var joinErr error
		runLobbyWorkers(
			func() { lobby.toggleReady("carol") },
			func() { _, joinErr = registry.Join(lobby.id, "dave") },
			func() { registry.List() },
		)
		if joinErr == nil {
			if row := registry.List()[0]; row.started || row.players != 4 {
				t.Fatalf("fourth join did not delay start: %+v", row)
			}
			lobby.toggleReady("dave")
			assertStartedPlayers(t, lobby, "alice", "bob", "carol", "dave")
		} else {
			assertStartedPlayers(t, lobby, "alice", "bob", "carol")
		}
	}
}

func TestLobbyConcurrentReadyAndUnreadyLeave(t *testing.T) {
	for attempt := 0; attempt < 40; attempt++ {
		registry := newLobbyRegistry()
		lobby := makeTestLobby(t, registry, "Leaving seat", "alice", "bob", "carol", "dave")
		lobby.toggleReady("alice")
		lobby.toggleReady("bob")
		runLobbyWorkers(
			func() { lobby.toggleReady("carol") },
			func() { registry.Leave(lobby, "dave") },
			func() {
				for i := 0; i < 16; i++ {
					registry.List()
				}
			},
		)
		assertStartedPlayers(t, lobby, "alice", "bob", "carol")
	}
}

func TestLobbyConcurrentDisconnectsRemoveRunningGame(t *testing.T) {
	registry := newLobbyRegistry()
	names := []string{"alice", "bob", "carol", "dave"}
	lobby := makeTestLobby(t, registry, "Disconnects", names...)
	for _, name := range names {
		lobby.toggleReady(name)
	}
	workers := []func(){func() {
		for i := 0; i < 32; i++ {
			registry.List()
		}
	}}
	for _, name := range names {
		workers = append(workers, func() { registry.Leave(lobby, name) })
	}
	runLobbyWorkers(workers...)
	if len(registry.List()) != 0 {
		t.Fatal("simultaneous disconnects retained an empty running game")
	}
}

func TestLobbyConcurrentSessionsBrowsersAndRegistryChurn(t *testing.T) {
	registry := newLobbyRegistry()
	var sessions []*SessionModel
	for group := 0; group < 4; group++ {
		var id uint64
		for player := 0; player < 4; player++ {
			session := newSessionModel(registry, fmt.Sprintf("player%d", player), 80, 32, nil)
			if player == 0 {
				createTestLobby(t, session, fmt.Sprintf("Game %d", group))
				id = session.lobby.id
			} else {
				session.Update(joinLobbyMsg{id: id})
			}
			sessions = append(sessions, session)
		}
	}

	backgroundWorkers := func() []func() {
		var workers []func()
		for i := 0; i < 4; i++ {
			workers = append(workers, func() {
				browser := newSessionModel(registry, fmt.Sprintf("browser%d", i), 80, 32, nil)
				for j := 0; j < 24; j++ {
					browser.Update(TickMsg{})
					browser.View()
				}
			})
			workers = append(workers, func() {
				for j := 0; j < 12; j++ {
					lobby, err := registry.Create(fmt.Sprintf("Temporary %d/%d", i, j), "host")
					if err != nil {
						t.Error(err)
						return
					}
					if _, err := registry.Join(lobby.id, "guest"); err != nil {
						t.Error(err)
					}
					lobby.toggleReady("guest")
					registry.List()
					registry.Leave(lobby, "host")
					registry.Leave(lobby, "guest")
				}
			})
		}
		return workers
	}

	workers := backgroundWorkers()
	for _, session := range sessions {
		workers = append(workers, func() {
			session.Update(tea.KeyMsg{Type: tea.KeyEnter})
			for i := 0; i < 8; i++ {
				session.Update(TickMsg{})
				session.View()
			}
		})
	}
	runLobbyWorkers(workers...)
	for _, session := range sessions {
		session.Update(TickMsg{})
		conn, ok := session.active.(*GameConn)
		if !ok || conn.userPlayer == nil || conn.game.Players[*conn.userPlayer].Name != session.username {
			t.Fatalf("%s lost its game or player assignment", session.username)
		}
	}

	// Game rendering/actions acquire game.m while other sessions read lobby
	// state, acquire registry locks, and disconnect from the same games.
	workers = backgroundWorkers()
	for _, session := range sessions {
		workers = append(workers, func() {
			for i := 0; i < 8; i++ {
				session.Update(tea.KeyMsg{Type: tea.KeyRight})
				session.Update(tea.KeyMsg{Type: tea.KeyEsc})
				session.View()
			}
			session.Close()
		})
	}
	runLobbyWorkers(workers...)
	if rows := registry.List(); len(rows) != 0 {
		t.Fatalf("sessions or temporary lobbies leaked after cleanup: %+v", rows)
	}
}
