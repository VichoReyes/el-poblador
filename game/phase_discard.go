package game

import (
	"fmt"
	"strings"

	"github.com/VichoReyes/el-poblador/board"
)

// discardRound coordinates all players before the roller may move the robber.
type discardRound struct {
	game    *Game
	pending map[int]bool
}

func startDiscardRound(g *Game) Phase {
	round := &discardRound{game: g, pending: make(map[int]bool)}
	var turn Phase
	for id := range g.Players {
		count := g.Players[id].TotalResources()
		var phase *phaseDiscard
		if count > 7 {
			round.pending[id] = true
			phase = newPhaseDiscard(round, id, count/2)
		} else {
			phase = newPhaseDiscard(round, id, 0)
		}
		if id == g.PlayerTurn {
			turn = phase
		} else {
			g.SetSpectatorPhase(id, phase)
		}
	}
	if len(round.pending) == 0 {
		return discardRobber(g)
	}
	return turn
}

type phaseDiscard struct {
	phaseWithOptions
	round     *discardRound
	remaining int
	invalid   string
}

func newPhaseDiscard(round *discardRound, id, count int) *phaseDiscard {
	options := make([]string, len(board.RESOURCE_TYPES))
	for i, resource := range board.RESOURCE_TYPES {
		options[i] = resource.String()
	}
	return &phaseDiscard{phaseWithOptions: phaseWithOptions{game: round.game, playerID: &id, options: options}, round: round, remaining: count}
}

func (p *phaseDiscard) HelpText() string {
	if p.remaining == 0 {
		if len(p.round.pending) == 0 {
			return "Waiting for the robber to be moved and any steal to finish."
		}
		return "Waiting for all players to discard before moving the robber."
	}
	if p.invalid != "" {
		return p.invalid
	}
	return fmt.Sprintf("Discard %d more resource card(s): select a resource and press Enter.", p.remaining)
}

func (p *phaseDiscard) Menu() RichText {
	if p.remaining == 0 {
		if len(p.round.pending) == 0 {
			return plainText("Waiting for robber")
		}
		var names []string
		for id := range p.game.Players {
			if p.round.pending[id] {
				names = append(names, p.game.Players[id].Name)
			}
		}
		return plainText("Waiting for discards: " + strings.Join(names, ", "))
	}
	options := make([]RichText, len(board.RESOURCE_TYPES))
	player := &p.game.Players[p.getPlayerID()]
	for i, resource := range board.RESOURCE_TYPES {
		options[i] = plainText(fmt.Sprintf("%s (%d)", resource, player.Resources[resource]))
	}
	return append(plainText(fmt.Sprintf("Discard %d more:\n", p.remaining)), p.menuOptions(options)...)
}

func (p *phaseDiscard) discard() bool {
	if p.remaining == 0 {
		return false
	}
	player := &p.game.Players[p.getPlayerID()]
	resource := board.RESOURCE_TYPES[p.selected]
	if player.Resources[resource] == 0 {
		p.invalid = "You have no " + resource.String() + " to discard."
		return false
	}
	p.invalid = ""
	player.Resources[resource]--
	p.remaining--
	if p.remaining == 0 {
		delete(p.round.pending, p.getPlayerID())
		p.game.LogAction(player, " finished discarding")
	}
	return len(p.round.pending) == 0
}

// resumeAfterDiscardRobber restores spectator actions only after the robber resolves.
type resumeAfterDiscardRobber struct{ Phase }

func discardRobber(g *Game) Phase {
	return PhasePlaceRobber(g, &resumeAfterDiscardRobber{PhaseIdle(g)})
}

func resumeRobber(g *Game, continuation Phase) Phase {
	if next, ok := continuation.(*resumeAfterDiscardRobber); ok {
		for id := range g.Players {
			if id != g.PlayerTurn {
				g.SetSpectatorPhase(id, PhaseSpectatorIdle(g, id))
			}
		}
		return next.Phase
	}
	return continuation
}

func (p *phaseDiscard) ConfirmTurn() (Phase, EnterMode) {
	if p.discard() {
		return discardRobber(p.game), EnterReset
	}
	return nil, EnterReset
}

func (p *phaseDiscard) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	if p.discard() {
		p.game.SetPhase(discardRobber(p.game))
		return nil, EnterReset
	}
	return nil, EnterReset
}
