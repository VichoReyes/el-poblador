package board

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestBoardPiecesAndCursorsUseViewerRenderer(t *testing.T) {
	b := NewDesertBoard()
	b.PlayerColors[0] = lipgloss.AdaptiveColor{Light: "#123456", Dark: "#FEDCBA"}
	cross := CrossCoord{X: 2, Y: 4}
	right := CrossCoord{X: 3, Y: 4}
	b.Settlements[cross], b.Settlements[right], b.CityUpgrades[right] = 0, 0, 0
	for _, end := range []CrossCoord{right, {X: 2, Y: 3}, {X: 2, Y: 5}} {
		b.Roads[NewPathCoord(cross, end)] = 0
	}
	cursor := CrossCoord{X: 3, Y: 5}
	tile := TileCoord{X: 2, Y: 3}
	for _, dark := range []bool{false, true} {
		r := lipgloss.NewRenderer(io.Discard)
		r.SetColorProfile(termenv.TrueColor)
		r.SetHasDarkBackground(dark)
		view := BoardView{Renderer: r, CrossCursor: &cursor, TileCursor: &tile}
		text := strings.Join(b.Print(view), "\n")
		color, cursorColor, background := "#123456", "#0277BD", "#E0E0E0"
		cursorForeground := "#FFFFFF"
		if dark {
			color, cursorColor, background = "#FEDCBA", "#4FC3F7", "#424242"
			cursorForeground = "#000000"
		}
		for _, piece := range []string{"▲▲▲", "███", "====", "//", "\\\\"} {
			if !strings.Contains(text, r.NewStyle().Foreground(lipgloss.Color(color)).Render(piece)) {
				t.Fatalf("piece %q did not use viewer palette", piece)
			}
		}
		if !strings.Contains(text, r.NewStyle().Foreground(lipgloss.Color(cursorForeground)).Background(lipgloss.Color(cursorColor)).Bold(true).Render(" ○ ")) {
			t.Fatal("crossing cursor did not use viewer palette")
		}
		terrainColor := "#795548"
		if dark {
			terrainColor = "#BCAAA4"
		}
		if !strings.Contains(text, r.NewStyle().Foreground(lipgloss.Color(terrainColor)).Background(lipgloss.Color(background)).Render("/‾‾‾‾\\")) {
			t.Fatal("tile cursor did not use viewer palette")
		}
		r.SetColorProfile(termenv.Ascii)
		if strings.Contains(strings.Join(b.Print(view), "\n"), "\x1b[") {
			t.Fatal("colorless viewer received styled board pieces")
		}
	}
}
