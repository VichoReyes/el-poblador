package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
)

type phaseRoadStart struct {
	game         *Game
	cursorCross  board.CrossCoord
	invalid      string
	isFree       bool
	continuation Phase
	helpPrefix   string
}

// Phase for building a road by paying for it
func PhaseRoadStart(game *Game) Phase {
	end := PhaseIdleWithNotification(game, "Road Built!")
	return newPhaseRoadStart(game, false, end, "")
}

// Phase for building two roads by using a development card
func PhaseRoadBuilding(game *Game) Phase {
	// First free road - continuation will be second free road
	end := PhaseIdleWithNotification(game, "Two free roads built!")
	second := newPhaseRoadStart(game, true, end, "second free")
	return newPhaseRoadStart(game, true, second, "first free")
}

func newPhaseRoadStart(game *Game, isFree bool, continuation Phase, helpPrefix string) Phase {
	cursorCross := game.Board.ValidCrossCoord()
	return &phaseRoadStart{
		game:         game,
		cursorCross:  cursorCross,
		isFree:       isFree,
		continuation: continuation,
		helpPrefix:   helpPrefix,
	}
}

func (p *phaseRoadStart) ConfirmTurn() (Phase, EnterMode) {
	// Check if player has a road or settlement connected to this crossing
	playerId := p.game.PlayerTurn
	if !p.game.Board.HasRoadConnected(p.cursorCross, playerId) {
		// Check if player has a settlement at this crossing
		if !p.game.Board.HasSettlementAt(p.cursorCross, playerId) {
			p.invalid = "You must have a road or settlement connected to this crossing"
			return p, EnterReplace // Invalid selection, stay in same phase
		}
	}
	next := newPhaseRoadEnd(p.game, p.cursorCross, p.isFree, p.continuation, p.helpPrefix)
	if p.isFree {
		return next, EnterReplace
	}
	return next, EnterPush
}

func (p *phaseRoadStart) HelpText() string {
	if p.invalid != "" {
		return p.invalid
	}
	if p.helpPrefix != "" {
		return fmt.Sprintf("Select the starting point for your %s road", p.helpPrefix)
	}
	return "Select the starting point for your road"
}

func (p *phaseRoadStart) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseRoadStart) MoveCursor(direction string) {
	dest, ok := moveCrossCursor(p.cursorCross, direction)
	if !ok {
		return
	}
	p.cursorCross = dest
}

func (p *phaseRoadStart) Menu() string {
	return "Select road starting point on the map"
}

type phaseRoadEnd struct {
	game         *Game
	startCross   board.CrossCoord
	cursorCross  board.CrossCoord
	invalid      string
	isFree       bool
	continuation Phase
	helpPrefix   string
}

func PhaseRoadEnd(game *Game, startCross board.CrossCoord) Phase {
	return newPhaseRoadEnd(game, startCross, false, PhaseIdleWithNotification(game, "Road Built!"), "")
}

func newPhaseRoadEnd(game *Game, startCross board.CrossCoord, isFree bool, continuation Phase, helpPrefix string) Phase {
	// Start with the first neighbor of the start cross
	neighbors := startCross.Neighbors()
	if len(neighbors) == 0 {
		panic("No neighbors found for start cross")
	}
	cursorCross := neighbors[0]

	return &phaseRoadEnd{
		game:         game,
		startCross:   startCross,
		cursorCross:  cursorCross,
		isFree:       isFree,
		continuation: continuation,
		helpPrefix:   helpPrefix,
	}
}

func (p *phaseRoadEnd) ConfirmTurn() (Phase, EnterMode) {
	player := &p.game.Players[p.game.PlayerTurn]
	playerId := p.game.PlayerTurn
	pathCoord := board.NewPathCoord(p.startCross, p.cursorCross)

	if !p.game.Board.CanPlaceRoad(pathCoord, playerId) {
		p.invalid = "Can't build road here"
		return p, EnterReplace
	}

	if player.RoadsAvailable <= 0 {
		return p.continuation, EnterReset
	}

	if !p.isFree {
		if !player.BuildRoad() {
			p.invalid = "Not enough resources"
			return p, EnterReplace
		}
	}

	p.game.Board.SetRoad(pathCoord, playerId)

	if p.isFree {
		p.game.LogAction(fmt.Sprintf("%s built a free road", player.RenderName()))
	} else {
		p.game.LogAction(fmt.Sprintf("%s built a road", player.RenderName()))
	}

	return p.continuation, EnterReset
}

func (p *phaseRoadEnd) HelpText() string {
	if p.invalid != "" {
		return p.invalid
	}
	if p.helpPrefix != "" {
		return fmt.Sprintf("Select the ending point for your %s road", p.helpPrefix)
	}
	return "Select the ending point for your road"
}

func (p *phaseRoadEnd) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseRoadEnd) MoveCursor(direction string) {
	var dest board.CrossCoord
	var ok bool

	switch direction {
	case "up":
		dest, ok = p.startCross.Up()
	case "down":
		dest, ok = p.startCross.Down()
	case "left":
		dest, ok = p.startCross.Left()
	case "right":
		dest, ok = p.startCross.Right()
	default:
		return
	}

	if ok {
		p.cursorCross = dest
	}
}

func (p *phaseRoadEnd) Menu() string {
	return "Select road ending point on the map"
}
