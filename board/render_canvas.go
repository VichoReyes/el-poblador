package board

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type boardFragment struct {
	x, y int
	text string
}

type boardCanvas struct {
	fragments              []boardFragment
	minX, minY, maxX, maxY int
}

func newBoardCanvas() *boardCanvas {
	// The land board occupies 50 columns and 31 rows, including building spaces.
	return &boardCanvas{maxX: 50, maxY: 31}
}

func (c *boardCanvas) write(x, y int, text string) {
	c.fragments = append(c.fragments, boardFragment{x, y, text})
	c.minX, c.minY = min(c.minX, x), min(c.minY, y)
	c.maxX, c.maxY = max(c.maxX, x+lipgloss.Width(text)), max(c.maxY, y+1)
}

// lines pads the complete board to its bounds, preserving the relative positions
// of every component. No styled content needs to be sliced or overwritten.
// It panics if fragments overlap.
func (c *boardCanvas) lines() []string {
	slices.SortFunc(c.fragments, func(a, b boardFragment) int {
		if a.y != b.y {
			return a.y - b.y
		}
		return a.x - b.x
	})
	lines := make([]string, c.maxY-c.minY)
	index := 0
	for y := c.minY; y < c.maxY; y++ {
		var line strings.Builder
		x := c.minX
		for index < len(c.fragments) && c.fragments[index].y == y {
			fragment := c.fragments[index]
			if fragment.x < x {
				panic(fmt.Sprintf("board canvas: overlapping fragments at column %d, row %d", fragment.x, y))
			}
			line.WriteString(strings.Repeat(" ", fragment.x-x))
			line.WriteString(fragment.text)
			x = fragment.x + lipgloss.Width(fragment.text)
			index++
		}
		line.WriteString(strings.Repeat(" ", c.maxX-x))
		lines[y-c.minY] = line.String()
	}
	return lines
}

// The board's staggered crossings are nine columns apart, with even crossings
// indented two columns, and three rows between successive Y coordinates.
func crossingPosition(coord CrossCoord) (int, int) {
	x := coord.X * 9
	if (coord.X+coord.Y)%2 == 0 {
		x += 2
	}
	return x, coord.Y * 3
}
