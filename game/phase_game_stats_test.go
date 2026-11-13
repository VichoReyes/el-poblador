package game

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func openGameStatsFromIdle(t *testing.T, g *Game, viewer *int) *phaseGameStats {
	t.Helper()
	var idle PhaseView = g.Phase()
	if viewer != nil && *viewer != g.PlayerTurn {
		idle, _ = g.SpectatorPhase(*viewer)
	}
	if idle == nil {
		t.Fatal("expected an idle menu")
	}
	for i := 0; i < 6; i++ {
		if strings.Contains(idle.Menu().String(), "> Game Stats") {
			g.ConfirmAction(viewer)
			if viewer == nil || *viewer == g.PlayerTurn {
				if info, ok := g.Phase().(*phaseGameStats); ok {
					return info
				}
			} else if phase, _ := g.SpectatorPhase(*viewer); phase != nil {
				if info, ok := phase.(*phaseGameStats); ok {
					return info
				}
			}
			t.Fatal("Game Stats menu item did not open the information phase")
		}
		g.MoveCursor("down", viewer)
	}
	t.Fatal("idle menu has no Game Stats option")
	return nil
}

func TestGameStatsRestoresTurnIdleMenu(t *testing.T) {
	g := awardGame()
	idle := PhaseIdleWithNotification(g, "Road Built!")
	g.SetPhase(idle)
	info := openGameStatsFromIdle(t, g, nil)
	g.ConfirmAction(nil)
	if g.Phase() != info || len(g.Board.Roads) != 0 {
		t.Fatal("confirming game stats must not perform a game action")
	}
	g.MoveCursor("down", nil)
	if info.selected != 1 {
		t.Fatal("down should select the next player's information")
	}
	g.CancelAction(nil)
	if g.Phase() != idle || !strings.Contains(idle.Menu().String(), "> Game Stats") || !strings.Contains(idle.HelpText(), "Road Built!") {
		t.Fatal("closing info did not restore the idle menu's selection and notification")
	}
	// End Turn remains reachable immediately after Game Stats.
	g.MoveCursor("down", nil)
	g.ConfirmAction(nil)
	if g.PlayerTurn != 1 {
		t.Fatal("End Turn did not advance to the next player")
	}
}

func TestGameStatsRestoresSpectatorIdleMenu(t *testing.T) {
	g := awardGame()
	viewer := 1
	idle := PhaseSpectatorIdle(g, viewer)
	g.SetSpectatorPhase(viewer, idle)
	turnPhase := g.Phase()
	info := openGameStatsFromIdle(t, g, &viewer)
	if info.selected != viewer {
		t.Fatal("info should open on the viewer")
	}
	g.MoveCursor("up", &viewer)
	g.ConfirmAction(&viewer)
	if info.selected != 0 || g.Phase() != turnPhase || len(g.TradeOffers) != 0 {
		t.Fatal("spectator info changed game state")
	}
	g.CancelAction(&viewer)
	if restored, _ := g.SpectatorPhase(viewer); restored != idle || !strings.Contains(idle.Menu().String(), "> Game Stats") {
		t.Fatal("closing info did not restore spectator's idle menu and selection")
	}
}

func TestGameStatsShowsPublicCardsAndAwards(t *testing.T) {
	g := awardGame()
	g.Players[1].PlayedDevCards = []DevCard{DevCardKnight, DevCardKnight, DevCardKnight, DevCardMonopoly, DevCardMonopoly, DevCardRoadBuilding}
	g.Players[1].HiddenDevCards = []DevCard{DevCardKnight, DevCardYearOfPlenty, DevCardVictoryPoint}
	roadLine(g, 1, 2, 0, 5)
	g.updateAwards()
	info := openGameStatsFromIdle(t, g, nil)
	if !strings.HasPrefix(info.Menu().String(), "Ana has played:\n  No development cards\n\n") {
		t.Fatal("player with no played development cards should show No development cards")
	}
	g.MoveCursor("down", nil)
	menu := info.Menu()
	for _, want := range []string{"Bob has played:", "Knight: 3", "Monopoly: 2", "Road Building: 1", "\n\nLongest Road: Bob (+2 VP)\nLargest Army: Bob (+2 VP)"} {
		if !strings.Contains(menu.String(), want) {
			t.Errorf("menu missing %q:\n%s", want, menu.String())
		}
	}
	for _, omitted := range []string{"Year of Plenty:", "Victory Point:", "No development cards", "Game stats", "unplayed dev cards", "Longest road length", "Ana", "Cat"} {
		if strings.Contains(menu.String(), omitted) {
			t.Errorf("menu should omit %q:\n%s", omitted, menu.String())
		}
	}
	if strings.Contains(menu.String(), "\x1b[") {
		t.Fatal("info menu contains pre-rendered styles")
	}
	for _, dark := range []bool{true, false} {
		r := viewerRenderer(termenv.TrueColor, dark)
		name := g.Players[1].RenderName(r)
		if !strings.Contains(menu.Render(r), name) {
			t.Fatal("game stats did not use the viewer's palette")
		}
	}
	plain := viewerRenderer(termenv.Ascii, true)
	for _, perspective := range []int{0, 2} {
		g.Players[perspective].HiddenDevCards = []DevCard{DevCardKnight, DevCardVictoryPoint}
		g.Players[perspective].PlayedDevCards = []DevCard{DevCardMonopoly}
		if perspective != 0 {
			g.SetSpectatorPhase(perspective, PhaseSpectatorIdle(g, perspective))
			openGameStatsFromIdle(t, g, &perspective)
		}
		view := g.Print(PrintOptions{Width: 34, Height: 32, Renderer: plain, RequestPlayer: &perspective})
		for _, want := range []string{"has played:", "Dice:", "Your resources:", "has 0 resources, 3 dev cards", "Dev Cards: 2", "Ana's turn."} {
			if !strings.Contains(view, want) {
				t.Errorf("shared sidebar missing %q:\n%s", want, view)
			}
		}
		if strings.Contains(view, "\x1b[") {
			t.Fatal("game stats leaked styles to a colorless viewer")
		}
		if lipgloss.Width(view) > 34 || lipgloss.Height(view) > 32 {
			t.Fatalf("game stats exceeds a 34x32 terminal: %dx%d", lipgloss.Width(view), lipgloss.Height(view))
		}
	}
}
