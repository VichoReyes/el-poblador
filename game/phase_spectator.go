package game

import (
	"el_poblador/board"
	"fmt"
	"strings"
)

// Spectator Phases
//
// This file implements phases for non-turn holders (spectators).
// Spectators can:
//   - Make trade offers to other players
//   - View and accept trade offers from the pool
//
// Spectator phases are separate from turn holder phases to prevent
// accidental mixing of concerns and provide type safety.

type phaseSpectatorIdle struct {
	game     *Game
	playerID int
	options  []string
	selected int
}

func PhaseSpectatorIdle(game *Game, playerID int) SpectatorPhase {
	return &phaseSpectatorIdle{
		game:     game,
		playerID: playerID,
		options: []string{
			"Make Trade Offer",
			"View Trade Pool",
		},
		selected: 0,
	}
}

func (p *phaseSpectatorIdle) MoveCursor(direction string) {
	switch direction {
	case "up":
		p.selected--
	case "down":
		p.selected++
	}
	p.selected = (p.selected + len(p.options)) % len(p.options)
}

func (p *phaseSpectatorIdle) Confirm(game *Game) SpectatorPhase {
	switch p.selected {
	case 0: // Make Trade Offer
		return PhaseSpectatorTradeOffer(game, p.playerID)
	case 1: // View Trade Pool
		return PhaseSpectatorTradePool(game, p.playerID)
	default:
		panic("Invalid option selected")
	}
}

func (p *phaseSpectatorIdle) Cancel(game *Game) SpectatorPhase {
	// Already at idle, nothing to cancel
	return p
}

func (p *phaseSpectatorIdle) HelpText() string {
	return "Waiting for your turn. You can make trade offers."
}

func (p *phaseSpectatorIdle) Menu() string {
	paddedOptions := make([]string, len(p.options))
	player := &p.game.Players[p.playerID]
	for i, option := range p.options {
		if i == p.selected {
			paddedOptions[i] = player.Render("> ") + option
		} else {
			paddedOptions[i] = "  " + option
		}
	}
	return strings.Join(paddedOptions, "\n")
}

// Helper base type for spectator phases with options
type spectatorPhaseWithOptions struct {
	game     *Game
	playerID int
	options  []string
	selected int
}

func (p *spectatorPhaseWithOptions) MoveCursor(direction string) {
	switch direction {
	case "up":
		p.selected--
	case "down":
		p.selected++
	}
	p.selected = (p.selected + len(p.options)) % len(p.options)
}

func (p *spectatorPhaseWithOptions) Menu() string {
	paddedOptions := make([]string, len(p.options))
	player := &p.game.Players[p.playerID]
	for i, option := range p.options {
		if i == p.selected {
			paddedOptions[i] = player.Render("> ") + option
		} else {
			paddedOptions[i] = fmt.Sprintf("  %s", option)
		}
	}
	return strings.Join(paddedOptions, "\n")
}

// phaseSpectatorTradePool - View and accept trade offers
type phaseSpectatorTradePool struct {
	spectatorPhaseWithOptions
	offersSnapshot []*TradeOffer
}

func PhaseSpectatorTradePool(game *Game, playerID int) SpectatorPhase {
	strikethrough := strikethroughStyle()

	// Categorize offers into actionable and non-actionable
	var actionable []*TradeOffer
	var nonActionable []*TradeOffer

	// Iterate in reverse (most recent first)
	for i := len(game.TradeOffers) - 1; i >= 0; i-- {
		offer := &game.TradeOffers[i]
		canTake := offer.canTake(playerID, game)

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
		options = append(options, offer.RenderForPool(game))
		snapshot = append(snapshot, offer)
	}

	// Add non-actionable offers (strikethrough)
	for _, offer := range nonActionable {
		options = append(options, strikethrough.Render(offer.RenderForPool(game)))
		snapshot = append(snapshot, offer)
	}

	// Always add Update button
	options = append(options, "Update")
	snapshot = append(snapshot, nil) // nil for Update button

	return &phaseSpectatorTradePool{
		spectatorPhaseWithOptions: spectatorPhaseWithOptions{
			game:     game,
			playerID: playerID,
			options:  options,
		},
		offersSnapshot: snapshot,
	}
}

func (p *phaseSpectatorTradePool) Confirm(game *Game) SpectatorPhase {
	selected := p.selectedOffer()
	if selected == nil {
		// "Update" button was selected - refresh the phase
		return PhaseSpectatorTradePool(game, p.playerID)
	}
	canTake := selected.canTake(p.playerID, game)

	switch canTake {
	case CanTakeOwnOffer:
		// Confirming an own offer means retracting it
		selected.Status = OfferIsRetracted
		return p
	case CanTakeExcludedPlayer:
		// Stay in trade pool - user can try another offer
		return p
	case CanTakeIsAmbiguous:
		// TODO: could open a counter-offer phase here
		// For now, stay in trade pool
		return p
	case CanTakeNotEnoughResources:
		// Stay in trade pool - user can try another offer
		return p
	case CanTakeObsolete:
		// Stay in trade pool - user can try another offer
		return p
	case CanTakeTrue:
		// Execute the trade
		if selected.executeTrade(p.playerID, game) {
			offerer := &game.Players[selected.OffererID]
			acceptor := &game.Players[p.playerID]
			game.LogAction(fmt.Sprintf("%s traded %s with %s",
				acceptor.RenderName(), selected.String(), offerer.RenderName()))
		}
		// Only successful trades return to idle
		return PhaseSpectatorIdle(game, p.playerID)
	default:
		// Stay in trade pool for unknown status
		return p
	}
}

func (p *phaseSpectatorTradePool) selectedOffer() *TradeOffer {
	if p.selected == len(p.offersSnapshot) {
		return nil
	}
	return p.offersSnapshot[p.selected]
}

func (p *phaseSpectatorTradePool) Cancel(game *Game) SpectatorPhase {
	return PhaseSpectatorIdle(game, p.playerID)
}

func (p *phaseSpectatorTradePool) HelpText() string {
	selected := p.selectedOffer()
	if selected == nil {
		return "Refresh offer list"
	}
	switch selected.canTake(p.playerID, p.game) {
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

// phaseSpectatorTradeOffer - Select resources to offer
type phaseSpectatorTradeOffer struct {
	game     *Game
	playerID int
	offer    map[board.ResourceType]int
	selected int
}

func PhaseSpectatorTradeOffer(game *Game, playerID int) SpectatorPhase {
	offer := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		offer[resourceType] = 0
	}

	return &phaseSpectatorTradeOffer{
		game:     game,
		playerID: playerID,
		offer:    offer,
		selected: 0,
	}
}

func (p *phaseSpectatorTradeOffer) MoveCursor(direction string) {
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
		player := &p.game.Players[p.playerID]
		maxAvailable := player.Resources[resourceType]
		if p.offer[resourceType] < maxAvailable {
			p.offer[resourceType]++
		}
	}
}

func (p *phaseSpectatorTradeOffer) Confirm(game *Game) SpectatorPhase {
	totalOffered := 0
	for _, amount := range p.offer {
		totalOffered += amount
	}
	if totalOffered == 0 {
		return p
	}
	return PhaseSpectatorTradeSelectReceive(game, p.playerID, p.offer, p)
}

func (p *phaseSpectatorTradeOffer) Cancel(game *Game) SpectatorPhase {
	return PhaseSpectatorIdle(game, p.playerID)
}

func (p *phaseSpectatorTradeOffer) Menu() string {
	var lines []string
	player := &p.game.Players[p.playerID]

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

func (p *phaseSpectatorTradeOffer) HelpText() string {
	return "Use ←/→ to adjust, ↑/↓ to move, Enter to confirm, Esc to cancel"
}

// phaseSpectatorTradeSelectReceive - Select resources to receive
type phaseSpectatorTradeSelectReceive struct {
	game          *Game
	playerID      int
	offer         map[board.ResourceType]int
	request       map[board.ResourceType]int
	previousPhase SpectatorPhase
	selected      int
}

func PhaseSpectatorTradeSelectReceive(game *Game, playerID int, offer map[board.ResourceType]int, offerPhase SpectatorPhase) SpectatorPhase {
	request := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		request[resourceType] = 0
	}

	return &phaseSpectatorTradeSelectReceive{
		game:          game,
		playerID:      playerID,
		offer:         offer,
		request:       request,
		previousPhase: offerPhase,
		selected:      0,
	}
}

func (p *phaseSpectatorTradeSelectReceive) MoveCursor(direction string) {
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

func (p *phaseSpectatorTradeSelectReceive) Confirm(game *Game) SpectatorPhase {
	totalRequested := 0
	for _, amount := range p.request {
		totalRequested += amount
	}
	if totalRequested == 0 {
		return p
	}

	// Spectators can only make player-to-player trades (not bank trades)
	// Create a broadcast offer (TargetID = -1)
	if game.AddTradeOffer(p.playerID, -1, p.offer, p.request) {
		offer := TradeOffer{
			Offering:   p.offer,
			Requesting: p.request,
		}
		player := &game.Players[p.playerID]
		game.LogAction(fmt.Sprintf("%s offered to trade %s",
			player.RenderName(), offer.String()))
		return PhaseSpectatorIdle(game, p.playerID)
	}

	return PhaseSpectatorIdle(game, p.playerID)
}

func (p *phaseSpectatorTradeSelectReceive) Cancel(game *Game) SpectatorPhase {
	return p.previousPhase
}

func (p *phaseSpectatorTradeSelectReceive) Menu() string {
	var lines []string
	player := &p.game.Players[p.playerID]

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

func (p *phaseSpectatorTradeSelectReceive) HelpText() string {
	return "Use ←/→ to adjust, ↑/↓ to move, Enter to confirm, Esc to cancel"
}
