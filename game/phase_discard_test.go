package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/VichoReyes/el-poblador/board"
)

func TestDiscardRoundWaitsForEveryPlayer(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C"}, false, true)
	resource := board.RESOURCE_TYPES[0]
	g.Players[0].Resources[resource] = 8
	g.Players[1].Resources[resource] = 9
	g.Players[2].Resources[resource] = 7
	g.SetPhase(startDiscardRound(g))
	spectator := 1
	other := 2
	g.CancelAction(nil)
	g.CancelAction(&spectator)
	g.ConfirmAction(&other)
	if _, ok := g.Phase().(*phaseDiscard); !ok {
		t.Fatal("roller bypassed discard")
	}
	for i := 0; i < 4; i++ {
		g.ConfirmAction(nil)
	}
	if g.Players[0].TotalResources() != 4 {
		t.Fatal("roller must discard half of 8")
	}
	if _, ok := g.Phase().(*phaseDiscard); !ok {
		t.Fatal("robber available before spectator discards")
	}
	for i := 0; i < 4; i++ {
		g.ConfirmAction(&spectator)
	}
	if g.Players[1].TotalResources() != 5 || g.Players[2].TotalResources() != 7 {
		t.Fatal("wrong discard totals")
	}
	robber, ok := g.Phase().(*phasePlaceRobber)
	if !ok {
		t.Fatal("robber unavailable after all discards")
	}
	if sp, _ := g.SpectatorPhase(spectator); !strings.Contains(sp.HelpText(), "Waiting") {
		t.Fatal("trading available before robber resolves")
	}
	for coord := range g.Board.Tiles {
		if coord != g.Board.GetRobber() {
			robber.tileCoord = coord
			break
		}
	}
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phaseIdle); !ok {
		t.Fatal("robber did not resolve")
	}
	if sp, _ := g.SpectatorPhase(spectator); sp == nil {
		t.Fatal("spectator idle missing")
	} else if _, ok := sp.(*phaseSpectatorIdle); !ok {
		t.Fatal("spectator actions not restored")
	}
}

func TestDiscardCannotSelectUnavailableResource(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C"}, false, true)
	g.Players[0].Resources[board.RESOURCE_TYPES[1]] = 8
	g.SetPhase(startDiscardRound(g))
	g.ConfirmAction(nil)
	if g.Players[0].TotalResources() != 8 {
		t.Fatal("discarded an unavailable resource")
	}
	p := g.Phase().(*phaseDiscard)
	if p.remaining != 4 {
		t.Fatal("unavailable resource counted toward discard")
	}
}

func TestDiscardLastTurnPlayerAndPrivateWaiting(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C"}, false, true)
	r := board.RESOURCE_TYPES[0]
	g.Players[0].Resources[r], g.Players[1].Resources[r] = 8, 8
	g.SetPhase(startDiscardRound(g))
	spectator := 1
	for i := 0; i < 4; i++ {
		g.ConfirmAction(&spectator)
	}
	sp, _ := g.SpectatorPhase(spectator)
	if !strings.Contains(sp.Menu().String(), "A") {
		t.Fatal("waiting menu must identify pending player")
	}
	if !strings.Contains(g.Phase().Menu().String(), "Discard 4 more") {
		t.Fatal("menu must show remaining count")
	}
	for i := 0; i < 4; i++ {
		g.ConfirmAction(nil)
	}
	if _, ok := g.Phase().(*phasePlaceRobber); !ok {
		t.Fatal("last turn player did not advance to robber")
	}
	sp, _ = g.SpectatorPhase(spectator)
	if !strings.Contains(sp.HelpText(), "robber") || !strings.Contains(sp.Menu().String(), "robber") {
		t.Fatal("waiting text must describe robber")
	}
	for _, entry := range g.ActionLog {
		if strings.Contains(entry.String(), r.String()) {
			t.Fatal("discard log reveals resource type")
		}
	}
}

func TestDiscardThresholdAndDevCards(t *testing.T) {
	for _, count := range []int{16, 17} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			g := &Game{}
			g.StartOptions([]string{"A", "B", "C"}, false, true)
			g.Players[1].Resources[board.RESOURCE_TYPES[0]] = count
			g.Players[0].HiddenDevCards = make([]DevCard, 10)
			g.SetPhase(startDiscardRound(g))
			if !strings.Contains(g.Phase().Menu().String(), "B") {
				t.Fatal("roller should wait for spectator")
			}
			spectator := 1
			for i := 0; i < count/2; i++ {
				g.ConfirmAction(&spectator)
			}
			if got := g.Players[1].TotalResources(); got != count-count/2 {
				t.Fatalf("resources left: %d", got)
			}
			if _, ok := g.Phase().(*phasePlaceRobber); !ok {
				t.Fatal("discard was repeated or development cards counted")
			}
		})
	}
}

func TestNoDiscardsWaitsForRobberAndSteal(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C"}, false, true)
	g.Players[1].Resources[board.RESOURCE_TYPES[0]] = 1
	g.SetPhase(startDiscardRound(g))
	spectator := 1
	g.CancelAction(&spectator)
	g.ConfirmAction(&spectator)
	if sp, _ := g.SpectatorPhase(spectator); !strings.Contains(sp.Menu().String(), "robber") {
		t.Fatal("spectator should wait for robber")
	}
	robber := g.Phase().(*phasePlaceRobber)
	g.Board.PlaceRobber(board.TileCoord{X: 2, Y: 5})
	g.Board.Settlements[board.CrossCoord{X: 2, Y: 3}] = spectator
	robber.tileCoord = board.TileCoord{X: 2, Y: 3}
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phaseStealCard); !ok {
		t.Fatal("expected steal phase")
	}
	g.ConfirmAction(&spectator)
	if sp, _ := g.SpectatorPhase(spectator); !strings.Contains(sp.Menu().String(), "robber") {
		t.Fatal("spectator could trade before steal")
	}
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phaseIdle); !ok {
		t.Fatal("steal did not end")
	}
	if g.Players[0].TotalResources() != 1 || g.Players[spectator].TotalResources() != 0 {
		t.Fatal("steal did not transfer the resource")
	}
	if sp, _ := g.SpectatorPhase(spectator); sp == nil {
		t.Fatal("missing spectator")
	} else if _, ok := sp.(*phaseSpectatorIdle); !ok {
		t.Fatal("spectator actions not restored")
	}
}

func TestKnightDoesNotRequireDiscards(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C"}, false, true)
	g.Players[0].Resources[board.RESOURCE_TYPES[0]] = 8
	g.Players[0].HiddenDevCards = []DevCard{DevCardKnight}
	g.SetPhase(PhaseDiceRoll(g))
	g.MoveCursor("down", nil)
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phasePlaceRobber); !ok {
		t.Fatal("knight must go directly to robber")
	}
	if g.Players[0].TotalResources() != 8 {
		t.Fatal("knight discarded resources")
	}
}

func TestFourPlayersChooseTheirOwnDiscards(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"A", "B", "C", "D"}, false, true)
	g.PlayerTurn = 2
	for id := range g.Players {
		g.Players[id].Resources[board.RESOURCE_TYPES[id]] = 3
		g.Players[id].Resources[board.RESOURCE_TYPES[id+1]] = 5
	}
	g.SetPhase(startDiscardRound(g))
	for step, id := range []int{1, 2, 0, 3} {
		for i := 0; i < id; i++ {
			g.MoveCursor("down", &id)
		}
		for i := 0; i < 3; i++ {
			g.ConfirmAction(&id)
		}
		g.MoveCursor("down", &id)
		g.ConfirmAction(&id)
		player := &g.Players[id]
		if player.Resources[board.RESOURCE_TYPES[id]] != 0 || player.Resources[board.RESOURCE_TYPES[id+1]] != 4 {
			t.Fatalf("player %d did not discard the chosen cards: %v", id, player.Resources)
		}
		if step < 3 {
			if _, ok := g.Phase().(*phaseDiscard); !ok {
				t.Fatal("robber available before every player discarded")
			}
		}
	}
	if _, ok := g.Phase().(*phasePlaceRobber); !ok {
		t.Fatal("robber unavailable after all four players discarded")
	}
}
