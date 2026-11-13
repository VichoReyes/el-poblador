package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
	"strings"
)

func nextInitialPhase(game *Game, isFirstPair bool) Phase {
	numPlayers := len(game.Players)

	if isFirstPair {
		if game.PlayerTurn == numPlayers-1 {
			return PhaseInitialSettlements(game, false)
		} else {
			game.PlayerTurn++
			return PhaseInitialSettlements(game, true)
		}
	} else {
		if game.PlayerTurn == 0 {
			return PhaseDiceRoll(game)
		} else {
			game.PlayerTurn--
			return PhaseInitialSettlements(game, false)
		}
	}
}

type phaseInitialSettlements struct {
	game        *Game
	cursorCross board.CrossCoord
	isFirstPair bool
}

func PhaseInitialSettlements(game *Game, isFirstPair bool) Phase {
	cursorCross := game.Board.ValidCrossCoord()
	return &phaseInitialSettlements{game: game, cursorCross: cursorCross, isFirstPair: isFirstPair}
}

func (p *phaseInitialSettlements) ConfirmTurn() (Phase, EnterMode) {

	player := &p.game.Players[p.game.PlayerTurn]

	if player.SettlementsAvailable <= 0 {
		p.game.LogAction("Player ", player, " has no settlements available during initial phase! Skipping...")
		return nextInitialPhase(p.game, p.isFirstPair), EnterReset
	}

	if !p.game.Board.SetSettlement(p.cursorCross, p.game.PlayerTurn) {
		return p, EnterReplace
	}

	if !p.isFirstPair {
		adjacentTiles := p.game.Board.AdjacentTiles(p.cursorCross)

		// Collect resources and track what was gained
		var resourcesGained []board.ResourceType
		for _, tile := range adjacentTiles {
			resource, ok := board.TileResource(tile)
			if ok {
				player.AddResource(resource)
				resourcesGained = append(resourcesGained, resource)
			}
		}

		// Log the initial resources gained
		if len(resourcesGained) > 0 {
			resourceCounts := make(map[board.ResourceType]int)
			for _, resource := range resourcesGained {
				resourceCounts[resource]++
			}

			var resourceParts []string
			for resourceType, count := range resourceCounts {
				resourceParts = append(resourceParts, fmt.Sprintf("%d %s", count, resourceType))
			}

			p.game.LogAction(player, " gained ", strings.Join(resourceParts, ", "), " from initial settlement")
		}
	}

	// Decrease available settlements after placing initial ones.
	player.SettlementsAvailable--
	p.game.updateAwards()

	// Check for game end after building initial settlement (unlikely but for completeness)
	if winner := p.game.CheckGameEnd(); winner != nil {
		return PhaseGameEnd(p.game, winner), EnterReset
	}

	return PhaseInitialRoad(p.game, p.cursorCross, p.isFirstPair), EnterReset
}

func (p *phaseInitialSettlements) HelpText() string {
	var num string
	if p.isFirstPair {
		num = "first"
	} else {
		num = "second"
	}
	return fmt.Sprintf("Place your %s settlement on the board with 'enter'.", num)
}

func (p *phaseInitialSettlements) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseInitialSettlements) MoveCursor(direction string) {
	dest, ok := moveCrossCursor(p.cursorCross, direction)
	if !ok {
		return
	}
	p.cursorCross = dest
}

func (p *phaseInitialSettlements) Menu() RichText {
	return plainText("Select initial settlement location on the map")
}

type phaseInitialRoad struct {
	game        *Game
	sourceCross board.CrossCoord
	cursorCross board.CrossCoord
	isFirstPair bool
}

func PhaseInitialRoad(game *Game, sourceCross board.CrossCoord, isFirstPair bool) Phase {
	return &phaseInitialRoad{
		game:        game,
		sourceCross: sourceCross,
		cursorCross: sourceCross.Neighbors()[0],
		isFirstPair: isFirstPair,
	}
}

func (p *phaseInitialRoad) ConfirmTurn() (Phase, EnterMode) {
	roadCoord := board.NewPathCoord(p.sourceCross, p.cursorCross)
	p.game.Board.SetRoad(roadCoord, p.game.PlayerTurn)
	p.game.Players[p.game.PlayerTurn].RoadsAvailable--
	p.game.updateAwards()
	return nextInitialPhase(p.game, p.isFirstPair), EnterReset
}

func (p *phaseInitialRoad) HelpText() string {
	return "Place a road connected to the settlement by selecting its direction."
}

func (p *phaseInitialRoad) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseInitialRoad) MoveCursor(direction string) {
	var dest board.CrossCoord
	var ok bool

	switch direction {
	case "up":
		dest, ok = p.sourceCross.Up()
	case "down":
		dest, ok = p.sourceCross.Down()
	case "left":
		dest, ok = p.sourceCross.Left()
	case "right":
		dest, ok = p.sourceCross.Right()
	default:
		return
	}

	if ok {
		p.cursorCross = dest
	}
}

func (p *phaseInitialRoad) Menu() RichText {
	return plainText("Select initial road direction on the map")
}
