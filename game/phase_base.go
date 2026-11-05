package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// when implementing, ensure the options aren't empty
type phaseWithOptions struct {
	game     *Game
	playerID *int // nil = use game.PlayerTurn (turn holder), non-nil = spectator
	options  []string
	selected int
}

func (p *phaseWithOptions) getPlayerID() int {
	if p.playerID != nil {
		return *p.playerID
	}
	return p.game.PlayerTurn
}

func (p *phaseWithOptions) BoardCursor() interface{} {
	return nil
}

func (p *phaseWithOptions) Menu() string {
	paddedOptions := make([]string, len(p.options))
	player := &p.game.Players[p.getPlayerID()]
	for i, option := range p.options {
		if i == p.selected {
			paddedOptions[i] = player.Render("> ") + option
		} else {
			paddedOptions[i] = fmt.Sprintf(" %s", option)
		}
	}
	return strings.Join(paddedOptions, "\n")
}

func (p *phaseWithOptions) MoveCursor(direction string) {
	switch direction {
	case "up":
		p.selected--
	case "down":
		p.selected++
	}
	p.selected = (p.selected + len(p.options)) % len(p.options)
}

func moveTileCursor(tileCoord board.TileCoord, direction string) (board.TileCoord, bool) {
	switch direction {
	case "up":
		return tileCoord.Up()
	case "down":
		return tileCoord.Down()
	case "left":
		return tileCoord.Left()
	case "right":
		return tileCoord.Right()
	default:
		return tileCoord, false
	}
}

func moveCrossCursor(cursorCross board.CrossCoord, direction string) (board.CrossCoord, bool) {
	switch direction {
	case "up":
		return cursorCross.Up()
	case "down":
		return cursorCross.Down()
	case "left":
		return cursorCross.Left()
	case "right":
		return cursorCross.Right()
	default:
		return cursorCross, false
	}
}

func strikethroughStyle() lipgloss.Style {
	return lipgloss.NewStyle().Strikethrough(true)
}
