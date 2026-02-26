package game

import (
	"strings"
)

// Spectator Phases
//
// This file implements the idle phase for non-turn holders (spectators).
// Spectators can:
//   - Make trade offers to other players
//   - View and accept trade offers from the pool
//
// The actual trade phases (phaseTradeOffer, phaseTradeSelectReceive, phaseTradePool)
// implement both Phase and SpectatorPhase interfaces, so they are shared between
// turn holders and spectators. See phase_trade.go.

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

func (p *phaseSpectatorIdle) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	switch p.selected {
	case 0: // Make Trade Offer
		return PhaseTradeOfferForPlayer(p.game, p.playerID), EnterPush
	case 1: // View Trade Pool
		return PhaseTradePoolForPlayer(p.game, p.playerID), EnterPush
	default:
		panic("Invalid option selected")
	}
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
