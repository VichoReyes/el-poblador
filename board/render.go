package board

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PrintBoard prints the game board made of ASCII hexagons
func (b *Board) Print(view BoardView) []string {
	// there will be 31 lines (5 * 5 + 6 for the roads)
	lines := make([]strings.Builder, 31)

	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			coord, valid := NewCrossCoord(x, y)
			if valid {
				renderCrossing(b, lines, coord, view)
			}
		}
	}

	renderedLines := []string{}
	for i := range lines {
		renderedLines = append(renderedLines, lines[i].String())
	}

	return sidePadding(renderedLines)
}

// takes responsibility for the crossing and whatever is to its right
// right-up and right-down paths
func renderCrossing(board *Board, lines []strings.Builder, coord CrossCoord, view BoardView) {
	// print crossing
	midLine := coord.Y * 3
	settlementOwner, hasSettlement := board.Settlements[coord]
	hasCursor := view.CrossCursor != nil && *view.CrossCursor == coord
	if hasCursor {
		cursorColor := lipgloss.AdaptiveColor{Light: "#0277BD", Dark: "#4FC3F7"}
		style := view.newStyle().Foreground(cursorColor).Blink(true)
		lines[midLine].WriteString(style.Render(" ○ "))
	} else if hasSettlement {
		_, isCity := board.CityUpgrades[coord]
		if isCity {
			lines[midLine].WriteString(renderPlayerContent(board.PlayerColors, settlementOwner, "███", view))
		} else {
			lines[midLine].WriteString(renderPlayerContent(board.PlayerColors, settlementOwner, "▲▲▲", view))
		}
	} else {
		lines[midLine].WriteString("   ")
	}

	// print right side
	if (coord.X+coord.Y)%2 == 1 {
		up, valid := coord.Up()
		if valid {
			path := NewPathCoord(coord, up)
			roadOwner, hasRoad := board.Roads[path]
			if hasRoad {
				lines[midLine-2].WriteString(renderPlayerContent(board.PlayerColors, roadOwner, "//", view))
				lines[midLine-1].WriteString(renderPlayerContent(board.PlayerColors, roadOwner, "//", view))
			} else {
				lines[midLine-2].WriteString("  ")
				lines[midLine-1].WriteString("  ")
			}
		}
		down, valid := coord.Down()
		if valid {
			path := NewPathCoord(coord, down)
			roadOwner, hasRoad := board.Roads[path]
			if hasRoad {
				lines[midLine+1].WriteString(renderPlayerContent(board.PlayerColors, roadOwner, "\\\\", view))
				lines[midLine+2].WriteString(renderPlayerContent(board.PlayerColors, roadOwner, "\\\\", view))
			} else {
				lines[midLine+1].WriteString("  ")
				lines[midLine+2].WriteString("  ")
			}
		}
		tileCoord, valid := NewTileCoord(coord.X, coord.Y)
		if valid {
			hasCursor := view.TileCursor != nil && *view.TileCursor == tileCoord
			tile := board.Tiles[tileCoord]
			hasRobber := board.Robber == tileCoord
			renderedTile := tile.RenderTile(hasCursor, hasRobber, view)
			lines[midLine-2].WriteString(renderedTile[0])
			lines[midLine-1].WriteString(renderedTile[1])
			lines[midLine].WriteString(renderedTile[2])
			lines[midLine+1].WriteString(renderedTile[3])
			lines[midLine+2].WriteString(renderedTile[4])
		}
	} else {
		right, valid := coord.Right()
		if !valid {
			return
		}
		pathCoord := NewPathCoord(coord, right)
		roadOwner, hasRoad := board.Roads[pathCoord]
		if hasRoad {
			lines[midLine].WriteString(renderPlayerContent(board.PlayerColors, roadOwner, "====", view))
		} else {
			lines[midLine].WriteString("    ")
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

// sidePadding centers the lines by padding to the widest one
func sidePadding(lines []string) []string {
	maxWidth := 0
	for _, line := range lines {
		width := lipgloss.Width(line)
		if width > maxWidth {
			maxWidth = width
		}
	}

	centered := make([]string, len(lines))
	for i, line := range lines {
		currentWidth := lipgloss.Width(line)
		if currentWidth < maxWidth {
			totalPadding := maxWidth - currentWidth
			leftPadding := totalPadding / 2
			rightPadding := totalPadding - leftPadding
			centered[i] = strings.Repeat(" ", leftPadding) + line + strings.Repeat(" ", rightPadding)
		} else {
			centered[i] = line
		}
	}

	return centered
}
