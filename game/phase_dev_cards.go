package game

import (
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
)

type phasePlayDevelopmentCard struct {
	phaseWithOptions
}

func PhasePlayDevelopmentCard(game *Game) Phase {
	player := &game.Players[game.PlayerTurn]

	// Build options based on available development cards
	var options []string

	for _, card := range player.HiddenDevCards {
		options = append(options, card.String())
	}

	options = append(options, "Cancel")

	return &phasePlayDevelopmentCard{
		phaseWithOptions: phaseWithOptions{
			game:    game,
			options: options,
		},
	}
}

func (p *phasePlayDevelopmentCard) ConfirmTurn() (Phase, EnterMode) {
	player := &p.game.Players[p.game.PlayerTurn]
	numCards := len(player.HiddenDevCards)
	if p.selected == numCards {
		return PhaseIdle(p.game), EnterReset
	}
	card := player.HiddenDevCards[p.selected]

	switch card {
	case DevCardKnight:
		player.PlayDevCard(card)
		p.game.LogAction(player, " played Knight")
		p.game.updateAwards()
		if winner := p.game.CheckGameEnd(); winner != nil {
			return PhaseGameEnd(p.game, winner), EnterReset
		}
		return PhasePlaceRobber(p.game, PhaseIdle(p.game)), EnterReset
	case DevCardRoadBuilding:
		player.PlayDevCard(card)
		p.game.LogAction(player, " played Road Building")
		return PhaseRoadBuilding(p.game), EnterReset
	case DevCardMonopoly:
		player.PlayDevCard(card)
		p.game.LogAction(player, " played Monopoly")
		return PhaseMonopoly(p.game), EnterReset
	case DevCardYearOfPlenty:
		player.PlayDevCard(card)
		p.game.LogAction(player, " played Year of Plenty")
		return PhaseYearOfPlenty(p.game), EnterReset
	case DevCardVictoryPoint:
		return PhaseIdle(p.game), EnterReset
	default:
		panic("This card does not exist")
	}
}

func (p *phasePlayDevelopmentCard) HelpText() string {
	return "Choose a development card to play"
}

type phaseMonopoly struct {
	phaseWithOptions
}

func PhaseMonopoly(game *Game) Phase {
	resourceOptions := make([]string, len(board.RESOURCE_TYPES))
	for i, resourceType := range board.RESOURCE_TYPES {
		resourceOptions[i] = resourceType.String()
	}
	resourceOptions = append(resourceOptions, "Cancel")

	return &phaseMonopoly{
		phaseWithOptions: phaseWithOptions{
			game:    game,
			options: resourceOptions,
		},
	}
}

func (p *phaseMonopoly) ConfirmTurn() (Phase, EnterMode) {
	if p.selected == len(board.RESOURCE_TYPES) {
		return PhaseIdle(p.game), EnterReset
	}

	selectedResource := board.RESOURCE_TYPES[p.selected]
	currentPlayer := p.game.Players[p.game.PlayerTurn]

	totalCollected := 0
	for i, player := range p.game.Players {
		if i != p.game.PlayerTurn {
			count := player.Resources[selectedResource]
			if count > 0 {
				player.Resources[selectedResource] = 0
				totalCollected += count
			}
		}
	}

	if totalCollected > 0 {
		currentPlayer.Resources[selectedResource] += totalCollected
		p.game.LogAction(currentPlayer, " collected ", totalCollected, " ", selectedResource, " from all players")
		return PhaseIdleWithNotification(p.game, fmt.Sprintf("Collected %d %s from other players!", totalCollected, selectedResource)), EnterReset
	} else {
		p.game.LogAction(currentPlayer, " monopolized ", selectedResource, " but collected nothing")
		return PhaseIdleWithNotification(p.game, "No resources collected - nobody had any!"), EnterReset
	}
}

func (p *phaseMonopoly) HelpText() string {
	return "Select a resource type to collect from all players"
}

type phaseYearOfPlenty struct {
	phaseWithOptions
	selectedCount     int
	selectedResources [2]board.ResourceType
}

func PhaseYearOfPlenty(game *Game) Phase {
	resourceOptions := make([]string, len(board.RESOURCE_TYPES))
	for i, resourceType := range board.RESOURCE_TYPES {
		resourceOptions[i] = resourceType.String()
	}
	resourceOptions = append(resourceOptions, "Cancel")

	return &phaseYearOfPlenty{
		phaseWithOptions: phaseWithOptions{
			game:    game,
			options: resourceOptions,
		},
		selectedCount: 0,
	}
}

func (p *phaseYearOfPlenty) ConfirmTurn() (Phase, EnterMode) {
	if p.selected == len(board.RESOURCE_TYPES) {
		return PhaseIdle(p.game), EnterReset
	}

	selectedResource := board.RESOURCE_TYPES[p.selected]
	p.selectedResources[p.selectedCount] = selectedResource
	p.selectedCount++

	if p.selectedCount < 2 {
		// Still need to select more resources
		return p, EnterReplace
	}

	// Both resources selected, give them to the player
	currentPlayer := &p.game.Players[p.game.PlayerTurn]
	for _, resource := range p.selectedResources {
		currentPlayer.AddResource(resource)
	}

	p.game.LogAction(currentPlayer, " gained ", p.selectedResources[0], " and ", p.selectedResources[1], " from the bank")

	return PhaseIdleWithNotification(p.game, fmt.Sprintf("Gained %s and %s from the bank!", p.selectedResources[0], p.selectedResources[1])), EnterReset
}

func (p *phaseYearOfPlenty) HelpText() string {
	if p.selectedCount == 0 {
		return "Select first resource to gain from the bank"
	}
	return fmt.Sprintf("Selected: %s. Select second resource to gain from the bank", p.selectedResources[0])
}
