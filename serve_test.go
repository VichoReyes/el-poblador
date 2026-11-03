package main

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

// The handler only needs the session's username. Unexpected SSH calls fail
// through the embedded nil interface instead of silently returning fake data.
type lobbyTestSession struct {
	ssh.Session
	username string
}

func (s lobbyTestSession) User() string { return s.username }

func TestLobbySessionsKeepPlayerNames(t *testing.T) {
	for _, playerCount := range []int{3, 4} {
		t.Run(fmt.Sprintf("%d_players", playerCount), func(t *testing.T) {
			// These cases share the server's global lobby and must run sequentially.
			originalLobby := singleLobby
			t.Cleanup(func() { singleLobby = originalLobby })

			// Exercise different map iteration orders without requiring a fixed
			// turn order or assuming it matches connection or readiness order.
			for attempt := 0; attempt < 20; attempt++ {
				singleLobby = &lobbyCommon{name: "Test lobby"}
				names := make([]string, playerCount)
				models := make([]tea.Model, playerCount)
				for i := range models {
					names[i] = fmt.Sprintf("name%d", i+1)
					models[i], _ = singleGameHandler(lobbyTestSession{username: names[i]})
					if models[i] == nil {
						t.Fatalf("attempt %d: %s could not join", attempt, names[i])
					}
				}

				for i := len(models) - 1; i >= 0; i-- {
					models[i], _ = models[i].Update(tea.KeyMsg{Type: tea.KeyEnter})
				}

				for i, model := range models {
					// Waiting sessions currently notice game start on a non-tick
					// message; a resize lets them finish the transition.
					model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
					conn, ok := model.(*UpdatingConn)
					if !ok {
						t.Fatalf("attempt %d: %s did not enter the game (model %T)", attempt, names[i], model)
					}
					if conn.userPlayer == nil {
						t.Fatalf("attempt %d: %s has no player assignment", attempt, names[i])
					}
					playerIndex := *conn.userPlayer
					if playerIndex < 0 || playerIndex >= len(conn.game.Players) {
						t.Fatalf("attempt %d: %s has invalid player index %d", attempt, names[i], playerIndex)
					}
					if got := conn.game.Players[playerIndex].Name; got != names[i] {
						t.Errorf("attempt %d: session %s became player %s", attempt, names[i], got)
					}
				}
			}
		})
	}
}
