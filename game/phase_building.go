package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
)

type phaseBuilding struct {
	phaseWithOptions
}

func PhaseBuilding(game *Game) Phase {
	player := &game.Players[game.PlayerTurn]

	// Build the list of available building options
	var options []string

	if player.CanBuildRoad() {
		options = append(options, fmt.Sprintf("Road - %d", player.RoadsAvailable))
	} else {
		options = append(options, "Road")
	}

	if player.CanBuildSettlement() {
		options = append(options, fmt.Sprintf("Settlement - %d", player.SettlementsAvailable))
	} else {
		options = append(options, "Settlement")
	}

	if player.CanBuildCity() {
		options = append(options, fmt.Sprintf("City - %d", player.CitiesAvailable))
	} else {
		options = append(options, "City")
	}

	if player.CanBuyDevelopmentCard() {
		options = append(options, "Development Card")
	} else {
		options = append(options, "Development Card")
	}

	options = append(options, "Cancel (or 'esc')")

	return &phaseBuilding{
		phaseWithOptions: phaseWithOptions{
			game:     game,
			options:  options,
			disabled: []bool{!player.CanBuildRoad(), !player.CanBuildSettlement(), !player.CanBuildCity(), !player.CanBuyDevelopmentCard()},
		},
	}
}

func (p *phaseBuilding) ConfirmTurn() (Phase, EnterMode) {
	player := &p.game.Players[p.game.PlayerTurn]

	switch p.selected {
	case 0: // Road
		if player.CanBuildRoad() {
			return PhaseRoadStart(p.game), EnterPush
		}
		return p, EnterReplace
	case 1: // Settlement
		if player.CanBuildSettlement() {
			return PhaseSettlementPlacement(p.game), EnterPush
		}
		return p, EnterReplace
	case 2: // City
		if player.CanBuildCity() {
			return PhaseCityPlacement(p.game), EnterPush
		}
		return p, EnterReplace
	case 3: // Development Card
		if player.CanBuyDevelopmentCard() {
			if player.BuyDevelopmentCard() {
				if card := p.game.DrawDevelopmentCard(); card != nil {
					player.HiddenDevCards = append(player.HiddenDevCards, *card)

					p.game.LogAction(player, " bought a development card")

					// Check for game end after buying development card (in case it's a victory point card)
					if winner := p.game.CheckGameEnd(); winner != nil {
						return PhaseGameEnd(p.game, winner), EnterReset
					}

					return PhaseIdleWithNotification(p.game, "Bought development card!"), EnterReset
				}
			}
		}
		return p, EnterReplace
	case 4: // Cancel
		return PhaseIdle(p.game), EnterReset
	default:
		panic("Invalid option selected")
	}
}

func (p *phaseBuilding) HelpText() string {
	return "Choose what to build"
}

type phaseSettlementPlacement struct {
	game        *Game
	cursorCross board.CrossCoord
	invalid     string
}

func PhaseSettlementPlacement(game *Game) Phase {
	cursorCross := game.Board.ValidCrossCoord()
	return &phaseSettlementPlacement{
		game:        game,
		cursorCross: cursorCross,
	}
}

func (p *phaseSettlementPlacement) ConfirmTurn() (Phase, EnterMode) {
	player := &p.game.Players[p.game.PlayerTurn]
	playerId := p.game.PlayerTurn

	if player.SettlementsAvailable == 0 {
		p.invalid = "No settlements left"
		return p, EnterReplace
	}

	if !p.game.Board.CanPlaceSettlementForPlayer(p.cursorCross, playerId) {
		p.invalid = "Can't build settlement here"
		return p, EnterReplace
	}

	if !player.BuildSettlement() {
		p.invalid = "Not enough resources"
		return p, EnterReplace
	}

	p.game.Board.SetSettlement(p.cursorCross, playerId)

	p.game.LogAction(player, " built a settlement")
	p.game.updateAwards()

	// Check for game end after building settlement
	if winner := p.game.CheckGameEnd(); winner != nil {
		return PhaseGameEnd(p.game, winner), EnterReset
	}

	return PhaseIdleWithNotification(p.game, "Settlement built!"), EnterReset
}

func (p *phaseSettlementPlacement) HelpText() string {
	if p.invalid != "" {
		return p.invalid
	}
	return "Select where to place your settlement"
}

func (p *phaseSettlementPlacement) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseSettlementPlacement) MoveCursor(direction string) {
	dest, ok := moveCrossCursor(p.cursorCross, direction)
	if !ok {
		return
	}
	p.cursorCross = dest
}

func (p *phaseSettlementPlacement) Menu() RichText {
	return plainText("Select settlement location on the map")
}

type phaseCityPlacement struct {
	game        *Game
	cursorCross board.CrossCoord
	invalid     string
}

func PhaseCityPlacement(game *Game) Phase {
	cursorCross := game.Board.ValidCrossCoord()
	return &phaseCityPlacement{
		game:        game,
		cursorCross: cursorCross,
	}
}

func (p *phaseCityPlacement) ConfirmTurn() (Phase, EnterMode) {
	player := &p.game.Players[p.game.PlayerTurn]
	playerId := p.game.PlayerTurn

	if player.CitiesAvailable == 0 {
		p.invalid = "No cities left"
		return p, EnterReplace
	}

	if !p.game.Board.CanUpgradeToCity(p.cursorCross, playerId) {
		p.invalid = "Can't upgrade to city here"
		return p, EnterReplace
	}

	if !player.BuildCity() {
		p.invalid = "Not enough resources"
		return p, EnterReplace
	}

	p.game.Board.UpgradeToCity(p.cursorCross, playerId)

	p.game.LogAction(player, " upgraded to a city")

	// Check for game end after building city
	if winner := p.game.CheckGameEnd(); winner != nil {
		return PhaseGameEnd(p.game, winner), EnterReset
	}

	return PhaseIdleWithNotification(p.game, "City built!"), EnterReset
}

func (p *phaseCityPlacement) HelpText() string {
	if p.invalid != "" {
		return p.invalid
	}
	return "Select a settlement to upgrade to a city"
}

func (p *phaseCityPlacement) BoardCursor() interface{} {
	return p.cursorCross
}

func (p *phaseCityPlacement) MoveCursor(direction string) {
	dest, ok := moveCrossCursor(p.cursorCross, direction)
	if !ok {
		return
	}
	p.cursorCross = dest
}

func (p *phaseCityPlacement) Menu() RichText {
	return plainText("Select city upgrade location on the map")
}
