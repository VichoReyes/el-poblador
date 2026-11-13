package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/VichoReyes/el-poblador/game"
)

func TestRemoteSaveResume(t *testing.T) {
	r := newLobbyRegistry()
	r.saves.dir = t.TempDir()
	l, err := r.Create("test", "alice")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bob", "carol", "dave"} {
		if _, err := r.Join(l.id, name); err != nil {
			t.Fatal(err)
		}
	}
	for name := range l.players {
		l.toggleReady(name)
	}
	original := make(map[string]int)
	for name, p := range l.players {
		original[name] = p.playerIndex
	}
	l.game.SetPhase(game.PhaseDiceRoll(&l.game))
	l.game.MoveCursor("down", nil)
	l.game.MoveCursor("down", nil)
	l.game.ConfirmAction(nil)
	if !l.game.ShouldQuit() || l.game.SaveNotice() == "" {
		t.Fatal("remote save did not pause with a code")
	}
	if !strings.Contains(l.game.SaveNotice(), l.code) {
		t.Fatal("missing code")
	}
	// A fresh registry simulates a server restart.
	resumedRegistry := newLobbyRegistry()
	resumedRegistry.saves.dir = r.saves.dir
	if _, err := resumedRegistry.Resume(l.code, "stranger"); err == nil {
		t.Fatal("accepted unknown player")
	}
	resumed, err := resumedRegistry.Resume(strings.ToLower(l.code), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(resumedRegistry.List()) != 0 {
		t.Fatal("saved room leaked into browser")
	}
	resumed.toggleReady("alice")
	for _, name := range []string{"bob", "carol"} {
		if _, err := resumedRegistry.Resume(l.code, name); err != nil {
			t.Fatal(err)
		}
		resumed.toggleReady(name)
	}
	if resumed.isStarted {
		t.Fatal("started without fourth player")
	}
	if _, err := resumedRegistry.Resume(l.code, "dave"); err != nil {
		t.Fatal(err)
	}
	resumed.toggleReady("dave")
	if !resumed.isStarted {
		t.Fatal("did not resume")
	}
	for name, index := range original {
		if resumed.players[name].playerIndex != index || resumed.game.Players[index].Name != name {
			t.Fatal("seat changed")
		}
	}
	if resumed.game.ShouldQuit() || resumed.game.SaveNotice() != "" {
		t.Fatal("restored paused flag")
	}
	if resumed.game.PlayerTurn != l.game.PlayerTurn {
		t.Fatal("turn changed")
	}
	// Re-saving replaces the snapshot under the same code.
	resumed.game.MoveCursor("down", nil)
	resumed.game.MoveCursor("down", nil)
	resumed.game.ConfirmAction(nil)
	if !resumed.game.ShouldQuit() {
		t.Fatal("resumed game could not save")
	}
	entries, _ := os.ReadDir(r.saves.dir)
	if len(entries) != 1 {
		t.Fatal("resaving created additional files")
	}
}

func TestSaveStorageLimitsAndExpiry(t *testing.T) {
	s := &saveStore{dir: t.TempDir()}
	var g game.Game
	g.MakeBoard()
	g.StartOptions([]string{"a", "b", "c"}, false, false)
	path := filepath.Join(s.dir, "old.gob")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxStoreBytes); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := s.write("0123456789ABCDEF", "test", &g); err == nil {
		t.Fatal("ignored quota")
	}
	old := time.Now().Add(-saveLifetime - time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := s.write("0123456789ABCDEF", "test", &g); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expired save retained")
	}
	for _, code := range []string{"../secret", "", "GGGGGGGGGGGGGGGG"} {
		if _, err := normalizeCode(code); err == nil {
			t.Fatal("accepted invalid code")
		}
	}
}

func TestFailedRemoteSaveKeepsGameOpen(t *testing.T) {
	r := newLobbyRegistry()
	// A regular file cannot be used as the save directory.
	path := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	r.saves.dir = path
	l, err := r.Create("test", "alice")
	if err != nil {
		t.Fatal(err)
	}
	l.game.StartOptions([]string{"alice", "bob", "carol"}, false, false)
	l.game.SetPhase(game.PhaseDiceRoll(&l.game))
	l.game.MoveCursor("down", nil)
	l.game.MoveCursor("down", nil)
	l.game.ConfirmAction(nil)
	if l.game.ShouldQuit() || l.game.SaveNotice() != "" {
		t.Fatal("failed save paused game")
	}
	if !strings.Contains(l.game.Phase().HelpText(), "Save failed") {
		t.Fatal("missing save error")
	}
	l.game.MoveCursor("up", nil)
	l.game.MoveCursor("up", nil)
	l.game.ConfirmAction(nil)
	if l.game.ShouldQuit() {
		t.Fatal("game no longer playable")
	}
}
