package board

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestPortsPreserveBoardOnEveryCoastalPath(t *testing.T) {
	b := NewDesertBoard()
	paths := make(map[PathCoord]bool)
	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			cross, ok := NewCrossCoord(x, y)
			if !ok {
				continue
			}
			b.Settlements[cross] = 0
			if y%3 == 0 {
				b.CityUpgrades[cross] = 0
			}
			for _, neighbor := range cross.Neighbors() {
				path := NewPathCoord(cross, neighbor)
				b.Roads[path] = 0
				if len(coastalTiles(path)) == 1 {
					paths[path] = true
				}
			}
		}
	}
	if len(paths) != 30 {
		t.Fatalf("got %d coastal paths, want 30", len(paths))
	}
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.Ascii)
	cursor := CrossCoord{X: 0, Y: 4}
	for _, emoji := range []bool{false, true} {
		view := BoardView{Renderer: r, UseEmoji: emoji, CrossCursor: &cursor}
		b.Ports = nil
		baseline := b.Print(view)
		for path := range paths {
			for _, resource := range []ResourceType{ResourceInvalid, ResourceWood} {
				t.Run(fmt.Sprintf("%v/emoji=%t/resource=%v", path, emoji, resource), func(t *testing.T) {
					b.Ports = map[PathCoord]ResourceType{path: resource}
					lines := b.Print(view)
					assertBoardSize(t, lines, 52, 33)
					anchor := "PT"
					if emoji {
						anchor = "⚓"
					}
					if strings.Count(strings.Join(lines, "\n"), anchor) != 1 {
						t.Fatal("expected exactly one port anchor")
					}
					// A left-edge port can add one column on the left; a top
					// horizontal port adds one row above the land board.
					dx, dy := 0, 0
					if path.From.X == 0 && path.To.X == 0 {
						dx = 1
					}
					if path.From.Y == 0 && path.To.Y == 0 {
						dy = 1
					}
					for y, before := range baseline {
						for x := 0; x < lipgloss.Width(before); x++ {
							cell := ansi.Cut(before, x, x+1)
							if strings.TrimSpace(cell) == "" {
								continue
							}
							if got := ansi.Cut(lines[y+dy], x+dx, x+dx+1); got != cell {
								t.Fatalf("port overwrote board at column %d, row %d: got %q, want %q", x, y, got, cell)
							}
						}
					}
				})
			}
		}
	}
}

func TestStandardPortRendering(t *testing.T) {
	b := NewLegalBoard(nil)
	for _, emoji := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			r := lipgloss.NewRenderer(io.Discard)
			r.SetColorProfile(termenv.TrueColor)
			r.SetHasDarkBackground(dark)
			view := BoardView{Renderer: r, UseEmoji: emoji}
			lines := b.Print(view)
			assertBoardSize(t, lines, 52, 33)
			if lipgloss.Width(lines[0]) != 52 || len(lines) != 32 {
				t.Fatalf("standard ports should produce a 52 × 32 board, got %d × %d", lipgloss.Width(lines[0]), len(lines))
			}
			text := strings.Join(lines, "\n")
			anchor := "PT"
			if emoji {
				anchor = "⚓"
			}
			if strings.Count(text, anchor) != 9 {
				t.Fatal("expected exactly nine port anchors")
			}
			color := "#1565C0"
			if dark {
				color = "#64B5F6"
			}
			style := r.NewStyle().Foreground(lipgloss.Color(color))
			if !strings.Contains(text, style.Render(anchor)) {
				t.Fatal("port did not use viewer palette")
			}
			if got := strings.Count(text, style.Render("3 ")) + strings.Count(text, style.Render(" 3")); got != 4 {
				t.Fatalf("got %d generic port labels, want 4", got)
			}
			for _, resource := range RESOURCE_TYPES {
				if got := strings.Count(text, style.Render(resource.Display(emoji))); got != 1 {
					t.Fatalf("got %d %s port labels, want 1", got, resource)
				}
			}
			r.SetColorProfile(termenv.Ascii)
			if strings.Contains(strings.Join(b.Print(view), "\n"), "\x1b[") {
				t.Fatal("colorless viewer received styled ports")
			}
		}
	}
}

func assertBoardSize(t *testing.T, lines []string, maxWidth, maxHeight int) {
	t.Helper()
	width := lipgloss.Width(lines[0])
	if width > maxWidth || len(lines) > maxHeight {
		t.Fatalf("board grew beyond %d × %d: got %d × %d", maxWidth, maxHeight, width, len(lines))
	}
	for y, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("row %d has width %d, want %d", y, got, width)
		}
	}
}
