package board

import (
	"slices"

	"github.com/charmbracelet/lipgloss"
)

func renderPort(canvas *boardCanvas, path PathCoord, resource ResourceType, view BoardView) {
	// A coastal path borders exactly one tile, which determines the land side.
	var land []TileCoord
	for _, tile := range path.From.adjacentTileCoords() {
		if slices.Contains(path.To.adjacentTileCoords(), tile) {
			land = append(land, tile)
		}
	}
	if len(land) != 1 {
		return
	}
	anchor := "PT"
	if view.UseEmoji {
		anchor = "⚓"
	}
	label := "3 "
	if resource != ResourceInvalid {
		label = resource.Display(view.UseEmoji)
	}
	style := view.newStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1565C0", Dark: "#64B5F6"})
	x, y := crossingPosition(path.From)
	if path.From.Y == path.To.Y {
		// Horizontal path: the two symbols fit directly above or below its four cells.
		outsideY := y - 1
		if land[0].Y < path.From.Y {
			outsideY = y + 1
		}
		canvas.write(x+3, outsideY, style.Render(label))
		canvas.write(x+5, outsideY, style.Render(anchor))
		return
	}
	// Diagonal paths occupy two cells on each of the two intervening rows.
	upperX, lowerX := x, x-1
	if (path.From.X+path.From.Y)%2 == 1 {
		upperX, lowerX = x+1, x+2
	}
	if land[0].X == path.From.X {
		upperX, lowerX = upperX-2, lowerX-2
		if resource == ResourceInvalid {
			label = " 3"
		}
	} else {
		upperX, lowerX = upperX+2, lowerX+2
	}
	canvas.write(upperX, y+1, style.Render(label))
	canvas.write(lowerX, y+2, style.Render(anchor))
}
