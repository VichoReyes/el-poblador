package game

import (
	"io"
	"strings"
	"testing"

	"github.com/VichoReyes/el-poblador/board"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func viewerRenderer(profile termenv.Profile, dark bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	r.SetHasDarkBackground(dark)
	return r
}

func TestSharedGameTextStyledAtRenderTime(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"Ána", "Ána María", "Wood"}, false, true)
	g.Players[1].Resources[board.ResourceWood] = 2
	g.Players[0].Resources[board.ResourceBrick] = 2
	g.AddTradeOffer(1, -1, map[board.ResourceType]int{board.ResourceWood: 1}, map[board.ResourceType]int{board.ResourceBrick: 1})
	pool := PhaseTradePool(g)
	building := PhaseBuilding(g)
	poolMenu, buildingMenu := pool.Menu(), building.Menu()
	for _, menu := range []RichText{poolMenu, buildingMenu} {
		if strings.Contains(menu.String(), "\x1b[") {
			t.Fatal("menu construction embedded terminal styles in its text")
		}
	}
	g.LogAction(&g.Players[0], " traded Wood with ", &g.Players[1])
	if got := g.ActionLog[0].String(); got != "Ána traded Wood with Ána María" {
		t.Fatalf("log should store plain text, got %q", got)
	}

	for _, dark := range []bool{false, true} {
		r := viewerRenderer(termenv.TrueColor, dark)
		color := "#C62828" // red player, light palette
		if dark {
			color = "#EF5350"
		}
		name := r.NewStyle().Foreground(lipgloss.Color(color)).Render("Ána María")
		if !strings.Contains(poolMenu.Render(r), name) {
			t.Fatal("trade label did not use viewer palette")
		}
		if !strings.Contains(g.renderActionLog(r), name) {
			t.Fatal("log did not use viewer palette")
		}
		if !strings.Contains(buildingMenu.Render(r), "\x1b[9m") {
			t.Fatal("disabled building option lost strikethrough")
		}
	}
	plain := viewerRenderer(termenv.Ascii, true)
	for _, text := range []string{poolMenu.Render(plain), buildingMenu.Render(plain), g.renderActionLog(plain)} {
		if strings.Contains(text, "\x1b[") {
			t.Fatal("cached game text leaked ANSI styles to a colorless viewer")
		}
	}
	if g.ActionLog[0].String() != "Ána traded Wood with Ána María" {
		t.Fatal("rendering mutated shared log text")
	}
}

func TestDiceRollKnightMenuStyle(t *testing.T) {
	g := &Game{}
	g.Start([]string{"Ana", "Bob", "Cat"})
	phase := PhaseDiceRoll(g).(*phaseDiceRoll)
	r := viewerRenderer(termenv.TrueColor, true)
	struckKnight := r.NewStyle().Strikethrough(true).Render("Play Knight")

	// Other players' Knights and already played Knights do not make this option available.
	g.Players[1].HiddenDevCards = []DevCard{DevCardKnight}
	g.Players[0].PlayedDevCards = []DevCard{DevCardKnight}
	if menu := phase.Menu().Render(r); !strings.Contains(menu, struckKnight) {
		t.Errorf("Knight option should be struck through without a hidden Knight: %q", menu)
	}
	phase.selected = 1
	if next, mode := phase.ConfirmTurn(); next != phase || mode != EnterReplace || phase.HelpText() != "You don't have a Knight card" {
		t.Fatal("Selecting an unavailable Knight should retain the existing error behavior")
	}

	g.Players[0].HiddenDevCards = []DevCard{DevCardKnight}
	if menu := phase.Menu().Render(r); strings.Contains(menu, struckKnight) || !strings.Contains(menu, "Play Knight") {
		t.Errorf("Knight option should be available with a hidden Knight: %q", menu)
	}

	g.SetPhase(phase)
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phasePlaceRobber); !ok {
		t.Fatal("Playing the last Knight should enter robber placement")
	}
	g.ConfirmAction(nil)
	if g.Phase() != phase {
		t.Fatal("Resolving the Knight should return to the same dice-roll phase")
	}
	if menu := phase.Menu().Render(r); !strings.Contains(menu, struckKnight) {
		t.Errorf("Knight option should be struck through after playing the last Knight: %q", menu)
	}
}

func TestGameEndMenu(t *testing.T) {
	g := &Game{}
	g.StartOptions([]string{"Ana", "Bob", "Cat"}, false, true)
	winner := &g.Players[1]
	phase := PhaseGameEnd(g, winner)
	menu := phase.Menu()
	if !strings.Contains(menu.String(), "Bob WINS!\n\nFinal Scores:") || !strings.Contains(menu.String(), "🏆 Bob:") {
		t.Fatalf("winner missing from game-end menu: %q", menu.String())
	}
	if got := phase.HelpText(); got != "Game complete! Bob reached 10 victory points!" {
		t.Fatalf("unexpected help: %q", got)
	}
	r := viewerRenderer(termenv.TrueColor, true)
	name := r.NewStyle().Foreground(winner.Color).Render(winner.Name)
	if !strings.Contains(menu.Render(r), name) {
		t.Fatal("winner name was not styled for viewer")
	}
	if strings.Contains(menu.String(), "\x1b[") {
		t.Fatal("plain menu contains ANSI styles")
	}
}
