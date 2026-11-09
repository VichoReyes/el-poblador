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
