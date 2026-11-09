package game

import (
	"fmt"

	"github.com/VichoReyes/el-poblador/board"
	"github.com/charmbracelet/lipgloss"
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
//   - Validation is deferred until complete trade is known (offer + request)
//   - Trade type detection happens in isBankTrade, isHarborTrade (future), etc.
//   - Both phases implement PhaseCancelable for Esc key handling

type phaseTradeOffer struct {
	game     *Game
	playerID *int // nil = use game.PlayerTurn (turn holder), non-nil = spectator
	offer    map[board.ResourceType]int
	selected int
}

// getPlayerID returns the player ID for this phase
func (p *phaseTradeOffer) getPlayerID() int {
	if p.playerID != nil {
		return *p.playerID
	}
	return p.game.PlayerTurn
}

func PhaseTradeOffer(game *Game) Phase {
	return newPhaseTradeOffer(game, nil)
}

func PhaseTradeOfferForPlayer(game *Game, playerID int) *phaseTradeOffer {
	return newPhaseTradeOffer(game, &playerID)
}

func newPhaseTradeOffer(game *Game, playerID *int) *phaseTradeOffer {
	offer := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		offer[resourceType] = 0
	}

	return &phaseTradeOffer{
		game:     game,
		playerID: playerID,
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
		player := &p.game.Players[p.getPlayerID()]
		maxAvailable := player.Resources[resourceType]
		if p.offer[resourceType] < maxAvailable {
			p.offer[resourceType]++
		}
	}
}

func (p *phaseTradeOffer) confirm() *phaseTradeSelectReceive {
	totalOffered := 0
	for _, amount := range p.offer {
		totalOffered += amount
	}
	if totalOffered == 0 {
		return nil
	}
	return newPhaseTradeSelectReceive(p.game, p.playerID, p.offer)
}

func (p *phaseTradeOffer) ConfirmTurn() (Phase, EnterMode) {
	if next := p.confirm(); next != nil {
		return next, EnterPush
	}
	return p, EnterReplace
}

// SpectatorPhase implementation
func (p *phaseTradeOffer) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	if next := p.confirm(); next != nil {
		return next, EnterPush
	}
	return p, EnterReplace
}

func (p *phaseTradeOffer) BoardCursor() interface{} {
	return nil
}

func (p *phaseTradeOffer) Menu() RichText {
	player := &p.game.Players[p.getPlayerID()]
	menu := plainText("What do you want to offer?\n")
	for i, resourceType := range board.RESOURCE_TYPES {
		menu = append(menu, TextPart{Text: "\n"})
		if i == p.selected {
			menu = append(menu, playerText(player, "> "))
		} else {
			menu = append(menu, TextPart{Text: "  "})
		}
		menu = append(menu, TextPart{Text: fmt.Sprintf("%s:  %d / %d", resourceType, p.offer[resourceType], player.Resources[resourceType])})
	}
	return menu
}

func (p *phaseTradeOffer) HelpText() string {
	return "Use ←/→ to adjust, ↑/↓ to move, Enter to confirm, Esc to cancel"
}

type phaseTradeSelectReceive struct {
	game     *Game
	playerID *int // nil = use game.PlayerTurn (turn holder), non-nil = spectator
	offer    map[board.ResourceType]int
	request  map[board.ResourceType]int
	selected int
}

// getPlayerID returns the player ID for this phase
func (p *phaseTradeSelectReceive) getPlayerID() int {
	if p.playerID != nil {
		return *p.playerID
	}
	return p.game.PlayerTurn
}

func PhaseTradeSelectReceive(game *Game, offer map[board.ResourceType]int, _ Phase) Phase {
	return newPhaseTradeSelectReceive(game, nil, offer)
}

func newPhaseTradeSelectReceive(game *Game, playerID *int, offer map[board.ResourceType]int) *phaseTradeSelectReceive {
	request := make(map[board.ResourceType]int)
	for _, resourceType := range board.RESOURCE_TYPES {
		request[resourceType] = 0
	}

	return &phaseTradeSelectReceive{
		game:     game,
		playerID: playerID,
		offer:    offer,
		request:  request,
		selected: 0,
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

func (p *phaseTradeSelectReceive) ConfirmTurn() (Phase, EnterMode) {
	totalRequested := 0
	for _, amount := range p.request {
		totalRequested += amount
	}
	if totalRequested == 0 {
		return p, EnterReplace
	}

	return p.validateAndExecuteTrade(), EnterReset
}

// SpectatorPhase implementation
func (p *phaseTradeSelectReceive) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	totalRequested := 0
	for _, amount := range p.request {
		totalRequested += amount
	}
	if totalRequested == 0 {
		return p, EnterReplace
	}

	return p.validateAndExecuteSpectatorTrade(), EnterReset
}

func (p *phaseTradeSelectReceive) BoardCursor() interface{} {
	return nil
}

func (p *phaseTradeSelectReceive) Menu() RichText {
	player := &p.game.Players[p.getPlayerID()]
	menu := plainText("What do you want to receive?\n")
	for i, resourceType := range board.RESOURCE_TYPES {
		menu = append(menu, TextPart{Text: "\n"})
		if i == p.selected {
			menu = append(menu, playerText(player, "> "))
		} else {
			menu = append(menu, TextPart{Text: "  "})
		}
		menu = append(menu, TextPart{Text: fmt.Sprintf("%s:  %d", resourceType, p.request[resourceType])})
	}
	return menu
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

		p.game.LogAction(player, " traded 4 ", offeredResource, " for 1 ", requestedResource, " with the bank")

		return PhaseIdleWithNotification(p.game,
			fmt.Sprintf("Traded 4 %s for 1 %s!", offeredResource, requestedResource))
	}

	// Player-to-player trade: create a broadcast offer (TargetID = -1)
	if p.game.AddTradeOffer(p.game.PlayerTurn, -1, p.offer, p.request) {
		offer := TradeOffer{
			Offering:   p.offer,
			Requesting: p.request,
		}
		p.game.LogAction(player, " offered to trade ", offer.String())
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

// validateAndExecuteSpectatorTrade handles trade execution for spectators (no bank trades)
func (p *phaseTradeSelectReceive) validateAndExecuteSpectatorTrade() SpectatorPhase {
	playerID := p.getPlayerID()
	player := &p.game.Players[playerID]

	// Spectators can only make player-to-player trades (not bank trades)
	// Create a broadcast offer (TargetID = -1)
	if p.game.AddTradeOffer(playerID, -1, p.offer, p.request) {
		offer := TradeOffer{
			Offering:   p.offer,
			Requesting: p.request,
		}
		p.game.LogAction(player, " offered to trade ", offer.String())
	}
	return PhaseSpectatorIdle(p.game, playerID)
}

type phaseTradePool struct {
	phaseWithOptions
	offersSnapshot  []*TradeOffer
	unavailableFrom int
}

// Phase for accepting offers made by non-turn holders.
func PhaseTradePool(g *Game) Phase {
	return newPhaseTradePool(g, nil)
}

func PhaseTradePoolForPlayer(g *Game, playerID int) *phaseTradePool {
	return newPhaseTradePool(g, &playerID)
}

func newPhaseTradePool(g *Game, playerID *int) *phaseTradePool {
	var currentPlayer int
	if playerID != nil {
		currentPlayer = *playerID
	} else {
		currentPlayer = g.PlayerTurn
	}

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

	// actionable offers
	for _, offer := range actionable {
		options = append(options, offer.String())
		snapshot = append(snapshot, offer)
	}

	// non-actionable offers (strikethrough)
	for _, offer := range nonActionable {
		options = append(options, offer.String())
		snapshot = append(snapshot, offer)
	}

	// Always add Update button
	options = append(options, "Update")
	snapshot = append(snapshot, nil) // nil for Update button

	return &phaseTradePool{
		phaseWithOptions: phaseWithOptions{
			game:     g,
			playerID: playerID,
			options:  options,
		},
		offersSnapshot:  snapshot,
		unavailableFrom: len(actionable),
	}
}

type poolConfirmOutcome int

const (
	poolConfirmRefresh poolConfirmOutcome = iota
	poolConfirmStay
	poolConfirmExit
)

func (p *phaseTradePool) confirmCommon() (poolConfirmOutcome, string) {
	selected := p.selectedOffer()
	if selected == nil {
		// "Update" button was selected - refresh the phase
		return poolConfirmRefresh, ""
	}
	playerID := p.getPlayerID()
	canTake := selected.canTake(playerID, p.game)

	switch canTake {
	case CanTakeOwnOffer:
		// Confirming an own offer means retracting it
		selected.Status = OfferIsRetracted
		return poolConfirmStay, "Offer retracted!"
	case CanTakeIsAmbiguous:
		// TODO: could open a counter-offer phase here
		// For now, stay in trade pool
		return poolConfirmStay, ""
	case CanTakeTrue:
		// Execute the trade
		offerer := &p.game.Players[selected.OffererID]
		acceptor := &p.game.Players[playerID]
		if selected.executeTrade(playerID, p.game) {
			p.game.LogAction(acceptor, " traded ", selected.String(), " with ", offerer)
			return poolConfirmExit, "Trade completed successfully!"
		}
		// Only reason the trade can fail at this point
		p.game.LogAction(offerer, "'s offer was a bluff!")
		selected.Status = OfferIsRetracted
		return poolConfirmExit, "Trade didn't work!"
	default:
		// Stay in trade pool for any other status
		return poolConfirmStay, ""
	}
}

func (p *phaseTradePool) ConfirmTurn() (Phase, EnterMode) {
	outcome, notification := p.confirmCommon()
	switch outcome {
	case poolConfirmRefresh:
		// Update button - refresh
		return PhaseTradePool(p.game), EnterReplace
	case poolConfirmStay:
		return p, EnterReplace
	case poolConfirmExit:
		return PhaseIdleWithNotification(p.game, notification), EnterReset
	default:
		panic("invalid pool confirm outcome")
	}
}

// SpectatorPhase implementation
func (p *phaseTradePool) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	outcome, _ := p.confirmCommon()
	switch outcome {
	case poolConfirmRefresh:
		// Update button - refresh
		return PhaseTradePoolForPlayer(p.game, p.getPlayerID()), EnterReplace
	case poolConfirmStay:
		return p, EnterReplace
	case poolConfirmExit:
		return PhaseSpectatorIdle(p.game, p.getPlayerID()), EnterReset
	default:
		panic("invalid pool confirm outcome")
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
	switch selected.canTake(p.getPlayerID(), p.game) {
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

// Keep offer ordering and selection in the phase, and style labels per viewer.
func (p *phaseTradePool) Menu() RichText {
	options := make([]RichText, len(p.options))
	for i, offer := range p.offersSnapshot {
		if offer == nil {
			options[i] = plainText("Update")
			continue
		}
		style := lipgloss.NewStyle().Strikethrough(i >= p.unavailableFrom)
		options[i] = offer.textForPool(p.game, style)
	}
	return p.phaseWithOptions.menuOptions(options)
}
