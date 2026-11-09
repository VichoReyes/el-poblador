package game

import (
	"github.com/VichoReyes/el-poblador/board"

	"github.com/charmbracelet/lipgloss"
)

// when implementing, ensure the options aren't empty
type phaseWithOptions struct {
	game     *Game
	playerID *int // nil = use game.PlayerTurn (turn holder), non-nil = spectator
	options  []string
	selected int
	disabled []bool
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

func (p *phaseWithOptions) Menu() RichText {
	options := make([]RichText, len(p.options))
	for i, option := range p.options {
		style := lipgloss.NewStyle().Strikethrough(i < len(p.disabled) && p.disabled[i])
		options[i] = RichText{{Text: option, style: style}}
	}
	return p.menuOptions(options)
}

func (p *phaseWithOptions) menuOptions(options []RichText) RichText {
	var menu RichText
	player := &p.game.Players[p.getPlayerID()]
	for i, option := range options {
		if i > 0 {
			menu = append(menu, TextPart{Text: "\n"})
		}
		if i == p.selected {
			menu = append(menu, playerText(player, "> "))
		} else {
			menu = append(menu, TextPart{Text: " "})
		}
		menu = append(menu, option...)
	}
	return menu
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
