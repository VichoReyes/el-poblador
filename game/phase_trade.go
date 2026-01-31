package game

import (
	"fmt"
	"git.sr.ht/~vicho/el_poblador/board"
	"strings"
)

// Trading System
//
// This file implements resource trading in El Poblador. The trading system uses a two-phase
// approach where players first specify what they want to offer, then what they want to receive.
// Validation only happens after the complete trade proposal is known, making the system extensible
// for different trade types.
//
// Phase Flow:
//   phaseIdle → phaseTradeOffer → phaseTradeSelectReceive → validateAndExecuteTrade → phaseIdle
//
// UI Controls:
//   - Up/Down: Navigate between resources
//   - Left/Right: Adjust resource amounts
//   - Enter: Confirm and proceed to next phase
//   - Esc: Cancel and return to previous phase
//
// Trade Types:
//   - Bank Trade (4:1): Offer exactly 4 of one resource for exactly 1 of another
//   - Harbor Trade (future): Offer 3:1 or 2:1 depending on harbor access
//   - Player Trade (future): Any combination of resources offered/requested
//
// Adding New Trade Types:
//   1. Add detection function (e.g., isHarborTrade) similar to isBankTrade
//   2. Add execution logic in validateAndExecuteTrade
//   3. For player trades, add new phases for partner selection and negotiation
//
// Design Notes:
//   - phaseTradeOffer has no previousPhase (always returns to phaseIdle)
//   - phaseTradeSelectReceive keeps reference to offer phase for cancel preservation
//   - Validation is deferred until complete trade is known (offer + request)
//   - Trade type detection happens in isBankTrade, isHarborTrade (future), etc.
//   - Both phases implement PhaseCancelable for Esc key handling

type phaseTradeOffer struct {
	game     *Game
	offer    map[board.ResourceType]int
	selected int
}

func PhaseTradeOffer(game *Game) Phase {
	offer := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		offer[resourceType] = 0
	}

	return &phaseTradeOffer{
		game:     game,
		offer:    offer,
		selected: 0,
	}
}

func (p *phaseTradeOffer) MoveCursor(direction string) {
	numResources := len(board.RESOURCE_TYPES)

	switch direction {
	case "up":
		p.selected--
		if p.selected < 0 {
			p.selected = numResources - 1
		}
	case "down":
		p.selected++
		if p.selected >= numResources {
			p.selected = 0
		}
	case "left":
		resourceType := board.RESOURCE_TYPES[p.selected]
		if p.offer[resourceType] > 0 {
			p.offer[resourceType]--
		}
	case "right":
		resourceType := board.RESOURCE_TYPES[p.selected]
		player := &p.game.Players[p.game.PlayerTurn]
		maxAvailable := player.Resources[resourceType]
		if p.offer[resourceType] < maxAvailable {
			p.offer[resourceType]++
		}
	}
}

func (p *phaseTradeOffer) Confirm() Phase {
	totalOffered := 0
	for _, amount := range p.offer {
		totalOffered += amount
	}
	if totalOffered == 0 {
		return p
	}
	return PhaseTradeSelectReceive(p.game, p.offer, p)
}

func (p *phaseTradeOffer) Cancel() Phase {
	return PhaseIdle(p.game)
}

func (p *phaseTradeOffer) BoardCursor() interface{} {
	return nil
}

func (p *phaseTradeOffer) Menu() string {
	var lines []string
	player := &p.game.Players[p.game.PlayerTurn]

	lines = append(lines, "What do you want to offer?")
	lines = append(lines, "")

	for i, resourceType := range board.RESOURCE_TYPES {
		amount := p.offer[resourceType]
		maxAvailable := player.Resources[resourceType]

		line := fmt.Sprintf("%s:  %d / %d", resourceType, amount, maxAvailable)
		if i == p.selected {
			line = player.Render("> ") + line
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (p *phaseTradeOffer) HelpText() string {
	return "Use ←/→ to adjust, ↑/↓ to move, Enter to confirm, Esc to cancel"
}

type phaseTradeSelectReceive struct {
	game          *Game
	offer         map[board.ResourceType]int
	request       map[board.ResourceType]int
	previousPhase Phase
	selected      int
}

func PhaseTradeSelectReceive(game *Game, offer map[board.ResourceType]int, offerPhase Phase) Phase {
	request := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		request[resourceType] = 0
	}

	return &phaseTradeSelectReceive{
		game:          game,
		offer:         offer,
		request:       request,
		previousPhase: offerPhase,
		selected:      0,
	}
}

func (p *phaseTradeSelectReceive) MoveCursor(direction string) {
	numResources := len(board.RESOURCE_TYPES)

	switch direction {
	case "up":
		p.selected--
		if p.selected < 0 {
			p.selected = numResources - 1
		}
	case "down":
		p.selected++
		if p.selected >= numResources {
			p.selected = 0
		}
	case "left":
		resourceType := board.RESOURCE_TYPES[p.selected]
		if p.request[resourceType] > 0 {
			p.request[resourceType]--
		}
	case "right":
		resourceType := board.RESOURCE_TYPES[p.selected]
		p.request[resourceType]++
	}
}

func (p *phaseTradeSelectReceive) Confirm() Phase {
	totalRequested := 0
	for _, amount := range p.request {
		totalRequested += amount
	}
	if totalRequested == 0 {
		return p
	}

	return p.validateAndExecuteTrade()
}

func (p *phaseTradeSelectReceive) Cancel() Phase {
	return p.previousPhase
}

func (p *phaseTradeSelectReceive) BoardCursor() interface{} {
	return nil
}

func (p *phaseTradeSelectReceive) Menu() string {
	var lines []string
	player := &p.game.Players[p.game.PlayerTurn]

	lines = append(lines, "What do you want to receive?")
	lines = append(lines, "")

	for i, resourceType := range board.RESOURCE_TYPES {
		amount := p.request[resourceType]

		line := fmt.Sprintf("%s:  %d", resourceType, amount)
		if i == p.selected {
			line = player.Render("> ") + line
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (p *phaseTradeSelectReceive) HelpText() string {
	return "Use ←/→ to adjust, ↑/↓ to move, Enter to confirm, Esc to cancel"
}

func (p *phaseTradeSelectReceive) validateAndExecuteTrade() Phase {
	player := &p.game.Players[p.game.PlayerTurn]

	if tradeType, offeredResource, requestedResource := p.isBankTrade(); tradeType == "bank" {
		if player.Resources[offeredResource] < 4 {
			return PhaseIdleWithNotification(p.game, "Not enough resources for bank trade!")
		}

		player.Resources[offeredResource] -= 4
		player.Resources[requestedResource]++

		p.game.LogAction(fmt.Sprintf("%s traded 4 %s for 1 %s with the bank",
			player.RenderName(), offeredResource, requestedResource))

		return PhaseIdleWithNotification(p.game,
			fmt.Sprintf("Traded 4 %s for 1 %s!", offeredResource, requestedResource))
	}

	// Player-to-player trade: create a broadcast offer (TargetID = -1)
	if p.game.AddTradeOffer(p.game.PlayerTurn, -1, p.offer, p.request) {
		offer := TradeOffer{
			Offering:   p.offer,
			Requesting: p.request,
		}
		p.game.LogAction(fmt.Sprintf("%s offered to trade %s",
			player.RenderName(), offer.String()))
		return PhaseIdleWithNotification(p.game, "Trade offer created!")
	}

	return PhaseIdleWithNotification(p.game, "Not enough resources to make this offer!")
}

func (p *phaseTradeSelectReceive) isBankTrade() (string, board.ResourceType, board.ResourceType) {
	totalOffered := 0
	offeredTypes := 0
	var offeredResource board.ResourceType
	for resourceType, amount := range p.offer {
		if amount > 0 {
			offeredTypes++
			offeredResource = resourceType
			totalOffered += amount
		}
	}

	totalRequested := 0
	requestedTypes := 0
	var requestedResource board.ResourceType
	for resourceType, amount := range p.request {
		if amount > 0 {
			requestedTypes++
			requestedResource = resourceType
			totalRequested += amount
		}
	}

	if totalOffered == 4 && offeredTypes == 1 && totalRequested == 1 && requestedTypes == 1 {
		return "bank", offeredResource, requestedResource
	}

	return "unknown", 0, 0
}

type phaseTradePool struct {
	phaseWithOptions
	offersSnapshot []*TradeOffer
}

// Phase for accepting offers made by non-turn holders.
// Remember only the holder has phases.
func PhaseTradePool(g *Game) Phase {
	currentPlayer := g.PlayerTurn
	strikethrough := strikethroughStyle()

	// Categorize offers into actionable and non-actionable
	var actionable []*TradeOffer
	var nonActionable []*TradeOffer

	// Iterate in reverse (most recent first)
	for i := len(g.TradeOffers) - 1; i >= 0; i-- {
		offer := &g.TradeOffers[i]
		canTake := offer.canTake(currentPlayer, g)

		if canTake == CanTakeTrue || canTake == CanTakeOwnOffer {
			actionable = append(actionable, offer)
		} else {
			nonActionable = append(nonActionable, offer)
		}
	}

	// Build options and snapshot in parallel
	var options []string
	var snapshot []*TradeOffer

	// Add actionable offers (normal style)
	for _, offer := range actionable {
		options = append(options, offer.RenderForPool(g))
		snapshot = append(snapshot, offer)
	}

	// Add non-actionable offers (strikethrough)
	for _, offer := range nonActionable {
		options = append(options, strikethrough.Render(offer.RenderForPool(g)))
		snapshot = append(snapshot, offer)
	}

	// Always add Update button
	options = append(options, "Update")
	snapshot = append(snapshot, nil) // nil for Update button

	return &phaseTradePool{
		phaseWithOptions: phaseWithOptions{
			game:    g,
			options: options,
		},
		offersSnapshot: snapshot,
	}
}

func (p *phaseTradePool) Confirm() Phase {
	selected := p.selectedOffer()
	if selected == nil {
		// "Update" button was selected - refresh the phase
		return PhaseTradePool(p.game)
	}
	canTake := selected.canTake(p.game.PlayerTurn, p.game)

	switch canTake {
	case CanTakeOwnOffer:
		// Confirming an own offer means retracting it
		selected.Status = OfferIsRetracted
		return p
	case CanTakeIsAmbiguous:
		// TODO: could open a counter-offer phase here
		// For now, stay in trade pool
		return p
	case CanTakeTrue:
		// Execute the trade
		offerer := &p.game.Players[selected.OffererID]
		acceptor := &p.game.Players[p.game.PlayerTurn]
		if selected.executeTrade(p.game.PlayerTurn, p.game) {
			p.game.LogAction(fmt.Sprintf("%s traded %s with %s",
				acceptor.RenderName(), selected.String(), offerer.RenderName()))
			return PhaseIdleWithNotification(p.game, "Trade completed successfully!")
		}
		// Only reason the trade can fail at this point
		p.game.LogAction(fmt.Sprintf("%s's offer was a bluff!", offerer.RenderName()))
		selected.Status = OfferIsRetracted
		return PhaseIdleWithNotification(p.game, "Trade didn't work!")
	default:
		// Stay in trade pool for any other status
		return p
	}
}

func (p *phaseTradePool) selectedOffer() *TradeOffer {
	if p.selected == len(p.offersSnapshot) {
		return nil
	}
	return p.offersSnapshot[p.selected]
}

func (p *phaseTradePool) HelpText() string {
	selected := p.selectedOffer()
	if selected == nil {
		return "Refresh offer list"
	}
	switch selected.canTake(p.game.PlayerTurn, p.game) {
	case CanTakeExcludedPlayer:
		return "Offer not for you"
	case CanTakeIsAmbiguous:
		return "Make counter offer to this?"
	case CanTakeOwnOffer:
		return "Your own offer. Retract?"
	case CanTakeNotEnoughResources:
		return "You don't have the necessary resources"
	case CanTakeObsolete:
		return "Offer no longer available"
	case CanTakeTrue:
		return "Accept this offer?"
	default:
		return "This should be impossible"
	}
}

func (p *phaseTradePool) Cancel() Phase {
	return PhaseIdle(p.game)
}
