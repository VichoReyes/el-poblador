package tui

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/VichoReyes/el-poblador/game"
	"github.com/charmbracelet/log"
)

func captureLifecycleLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	previous := log.Default()
	output := &bytes.Buffer{}
	log.SetDefault(log.NewWithOptions(output, log.Options{Formatter: log.JSONFormatter}))
	t.Cleanup(func() { log.SetDefault(previous) })
	return output
}

func events(t *testing.T, output *bytes.Buffer, message string) []map[string]any {
	t.Helper()
	var matches []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n")) {
		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		if event["msg"] == message {
			matches = append(matches, event)
		}
	}
	return matches
}

func TestLifecycleLogging(t *testing.T) {
	output := captureLifecycleLogs(t)
	r := newLobbyRegistry()
	r.saves.dir = t.TempDir()
	l, err := r.Create("saved room", "alice")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bob", "carol"} {
		if _, err := r.Join(l.id, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"alice", "bob", "carol"} {
		l.toggleReady(name)
	}
	l.game.SetPhase(game.PhaseDiceRoll(&l.game))
	l.game.MoveCursor("down", nil)
	l.game.MoveCursor("down", nil)
	l.game.ConfirmAction(nil)
	r2 := newLobbyRegistry()
	r2.saves.dir = r.saves.dir
	var resumed *lobbyCommon
	for _, name := range []string{"alice", "bob", "carol"} {
		resumed, err = r2.Resume(l.code, name)
		if err != nil {
			t.Fatal(err)
		}
		resumed.toggleReady(name)
	}
	// Ending the turn hands play to a player with ten points, triggering a win.
	winner := &resumed.game.Players[1]
	for i := 0; i < 10; i++ {
		winner.HiddenDevCards = append(winner.HiddenDevCards, game.DevCardVictoryPoint)
	}
	resumed.game.SetPhase(game.PhaseIdle(&resumed.game))
	resumed.game.MoveCursor("up", nil)
	resumed.game.ConfirmAction(nil)
	for i := 0; i < 3; i++ {
		resumed.game.ConfirmAction(nil)
	}
	for _, message := range []string{"game started", "game saved", "saved game loaded", "game resumed", "game finished"} {
		records := events(t, output, message)
		if len(records) != 1 {
			t.Fatalf("%s logged %d times: %s", message, len(records), output)
		}
		if records[0]["level"] != "info" || records[0]["lobby"] != "saved room" {
			t.Fatalf("missing event context: %v", records[0])
		}
	}
	finished := events(t, output, "game finished")[0]
	if finished["winner"] != winner.Name {
		t.Fatalf("wrong winner: %v", finished)
	}
	scores := finished["scores"].(map[string]any)
	if scores[winner.Name] != float64(10) || len(scores) != 3 {
		t.Fatalf("wrong scores: %v", scores)
	}
	saved := events(t, output, "game saved")[0]
	if saved["code"] != l.code || saved["bytes"].(float64) <= 0 || saved["expires_at"] == nil {
		t.Fatalf("missing save context: %v", saved)
	}
}
