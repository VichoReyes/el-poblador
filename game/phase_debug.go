package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
)

type phaseDebugTiles struct {
	game      *Game
	tileCoord board.TileCoord
}

func PhaseDebugTiles(game *Game) Phase {
	return &phaseDebugTiles{
		game:      game,
		tileCoord: game.Board.ValidTileCoord(),
	}
}

func (p *phaseDebugTiles) BoardCursor() interface{} {
	return p.tileCoord
}

func (p *phaseDebugTiles) MoveCursor(direction string) {
	dest, ok := moveTileCursor(p.tileCoord, direction)
	if !ok {
		return
	}
	p.tileCoord = dest
}

func (p *phaseDebugTiles) HelpText() string {
	return fmt.Sprintf("Tile x=%v, y=%v", p.tileCoord.X, p.tileCoord.Y)
}

func (p *phaseDebugTiles) Menu() RichText {
	return plainText(p.HelpText())
}

func (p *phaseDebugTiles) ConfirmTurn() (Phase, EnterMode) {
	newP := &phaseDebugCross{
		game: p.game,
		crossCoord: board.CrossCoord{
			X: p.tileCoord.X,
			Y: p.tileCoord.Y,
		},
	}
	return newP, EnterReplace
}

type phaseDebugCross struct {
	game       *Game
	crossCoord board.CrossCoord
}

func (p *phaseDebugCross) BoardCursor() interface{} {
	return p.crossCoord
}

func (p *phaseDebugCross) MoveCursor(direction string) {
	dest, ok := moveCrossCursor(p.crossCoord, direction)
	if !ok {
		return
	}
	p.crossCoord = dest
}

func (p *phaseDebugCross) HelpText() string {
	return fmt.Sprintf("Crossing x=%v, y=%v", p.crossCoord.X, p.crossCoord.Y)
}

func (p *phaseDebugCross) Menu() RichText {
	return plainText(p.HelpText())
}

func (p *phaseDebugCross) ConfirmTurn() (Phase, EnterMode) {
	newP := PhaseDebugTiles(p.game)
	return newP, EnterReplace
}
