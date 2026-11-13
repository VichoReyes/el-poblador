package board

import "github.com/charmbracelet/lipgloss"

// Print renders the game board made of ASCII hexagons.
func (b *Board) Print(view BoardView) []string {
	// Tiles, pieces, and ports all use the same board coordinates. Padding is
	// calculated once, after every component has contributed to the bounds.
	canvas := newBoardCanvas()

	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			coord, valid := NewCrossCoord(x, y)
			if valid {
				renderCrossing(b, canvas, coord, view)
			}
		}
	}

	for path, resource := range b.Ports {
		renderPort(canvas, path, resource, view)
	}

	return canvas.lines()
}

// takes responsibility for the crossing and whatever is to its right
// right-up and right-down paths
func renderCrossing(board *Board, canvas *boardCanvas, coord CrossCoord, view BoardView) {
	// print crossing
	x, midLine := crossingPosition(coord)
	settlementOwner, hasSettlement := board.Settlements[coord]
	hasCursor := view.CrossCursor != nil && *view.CrossCursor == coord
	content := " ○ "
	if hasSettlement {
		content = "▲▲▲"
		if _, isCity := board.CityUpgrades[coord]; isCity {
			content = "███"
		}
	}
	if hasCursor {
		// A contrasting background keeps both the marker and any building visible.
		style := view.newStyle().Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}).
			Background(lipgloss.AdaptiveColor{Light: "#0277BD", Dark: "#4FC3F7"}).Bold(true)
		canvas.write(x, midLine, style.Render(content))
	} else if hasSettlement {
		canvas.write(x, midLine, renderPlayerContent(board.PlayerColors, settlementOwner, content, view))
	}

	// print right side
	if (coord.X+coord.Y)%2 == 1 {
		up, valid := coord.Up()
		if valid {
			path := NewPathCoord(coord, up)
			roadOwner, hasRoad := board.Roads[path]
			if hasRoad {
				canvas.write(x+2, midLine-2, renderPlayerContent(board.PlayerColors, roadOwner, "//", view))
				canvas.write(x+1, midLine-1, renderPlayerContent(board.PlayerColors, roadOwner, "//", view))
			}
		}
		down, valid := coord.Down()
		if valid {
			path := NewPathCoord(coord, down)
			roadOwner, hasRoad := board.Roads[path]
			if hasRoad {
				canvas.write(x+1, midLine+1, renderPlayerContent(board.PlayerColors, roadOwner, "\\\\", view))
				canvas.write(x+2, midLine+2, renderPlayerContent(board.PlayerColors, roadOwner, "\\\\", view))
			}
		}
		tileCoord, valid := NewTileCoord(coord.X, coord.Y)
		if valid {
			hasCursor := view.TileCursor != nil && *view.TileCursor == tileCoord
			tile := board.Tiles[tileCoord]
			hasRobber := board.Robber == tileCoord
			renderedTile := tile.RenderTile(hasCursor, hasRobber, view)
			canvas.write(x+4, midLine-2, renderedTile[0])
			canvas.write(x+3, midLine-1, renderedTile[1])
			canvas.write(x+3, midLine, renderedTile[2])
			canvas.write(x+3, midLine+1, renderedTile[3])
			canvas.write(x+4, midLine+2, renderedTile[4])
		}
	} else {
		right, valid := coord.Right()
		if !valid {
			return
		}
		pathCoord := NewPathCoord(coord, right)
		roadOwner, hasRoad := board.Roads[pathCoord]
		if hasRoad {
			canvas.write(x+3, midLine, renderPlayerContent(board.PlayerColors, roadOwner, "====", view))
		}
	}
}

// renderPlayerContent applies player color to content string
func renderPlayerContent(playerColors map[int]lipgloss.AdaptiveColor, playerId int, content string, view BoardView) string {
	if color, ok := playerColors[playerId]; ok {
		style := view.newStyle().Foreground(color)
		return style.Render(content)
	}
	return content
}
