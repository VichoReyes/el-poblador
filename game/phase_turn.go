package game

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/VichoReyes/el-poblador/board"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"time"
)

type phaseDiceRoll struct {
	phaseWithOptions
	invalid string
}

func PhaseDiceRoll(game *Game) Phase {
	return &phaseDiceRoll{
		phaseWithOptions: phaseWithOptions{
			game:    game,
			options: []string{"Roll", "Play Knight", "Save & Quit"},
		},
	}
}

func (p *phaseDiceRoll) Menu() RichText {
	options := p.phaseWithOptions
	player := &p.game.Players[p.game.PlayerTurn]
	options.disabled = []bool{false, !slices.Contains(player.HiddenDevCards, DevCardKnight)}
	return options.Menu()
}

func (p *phaseDiceRoll) ConfirmTurn() (Phase, EnterMode) {
	switch p.selected {
	case 0:
		return rollDice(p.game), EnterReset
	case 1:
		// Play Knight card
		player := &p.game.Players[p.game.PlayerTurn]
		if player.PlayDevCard(DevCardKnight) {
			p.game.LogAction(player, " played Knight")
			p.game.updateAwards()
			if winner := p.game.CheckGameEnd(); winner != nil {
				return PhaseGameEnd(p.game, winner), EnterReset
			}
			return PhasePlaceRobber(p.game, p), EnterReset
		}
		p.invalid = "You don't have a Knight card"
		return p, EnterReplace
	case 2:
		// Save & Quit
		if err := p.game.save(); err != nil {
			p.invalid = fmt.Sprintf("Save failed: %v", err)
			return p, EnterReplace
		}
		p.game.shouldQuit = true
		return p, EnterReplace
	default:
		panic("Invalid option selected")
	}
}

func (p *phaseDiceRoll) HelpText() string {
	if p.invalid != "" {
		return p.invalid
	}
	return "Time to roll the dice"
}

func rollDice(game *Game) Phase {
	game.LastDice = [2]int{rand.IntN(6) + 1, rand.IntN(6) + 1}
	sum := game.LastDice[0] + game.LastDice[1]

	// Initialize spectator phases for all non-turn holders
	game.spectatorPhaseStacks = make(map[int][]SpectatorPhase)
	for i := range game.Players {
		if i != game.PlayerTurn {
			game.SetSpectatorPhase(i, PhaseSpectatorIdle(game, i))
		}
	}

	if sum == 7 {
		return startDiscardRound(game)
	}
	generatedResources := game.Board.GenerateResources(sum)
	for player, resources := range generatedResources {
		for _, r := range resources {
			game.Players[player].AddResource(r)
		}
	}

	// Log resource generation for each player if they received any
	for playerId, resources := range generatedResources {
		if len(resources) > 0 {
			// Count resources by type
			resourceCounts := make(map[board.ResourceType]int)
			for _, resource := range resources {
				resourceCounts[resource]++
			}

			// Build resource description
			var resourceParts []string
			for resourceType, count := range resourceCounts {
				resourceParts = append(resourceParts, fmt.Sprintf("%d %s", count, resourceType))
			}

			game.LogAction(&game.Players[playerId], " gained ", strings.Join(resourceParts, ", "), " from dice roll (", sum, ")")
		}
	}
	return PhaseIdle(game)
}

type phaseIdle struct {
	phaseWithOptions
	notification string
}

func PhaseIdle(game *Game) Phase {
	return PhaseIdleWithNotification(game, "")
}

func PhaseIdleWithNotification(game *Game, notification string) Phase {
	return &phaseIdle{
		phaseWithOptions: phaseWithOptions{
			game: game,
			options: []string{
				"Build",
				"Make Trade Offer",
				"Check Trade Offers",
				"Play Development Card",
				"Game Stats",
				"End Turn",
			},
		},
		notification: notification,
	}
}

func (p *phaseIdle) ConfirmTurn() (Phase, EnterMode) {
	switch p.selected {
	case 0: // Build
		return PhaseBuilding(p.game), EnterPush
	case 1: // Make Trade Offer
		return PhaseTradeOffer(p.game), EnterPush
	case 2: // Check Trade Offers
		return PhaseTradePool(p.game), EnterPush
	case 3: // Play Development Card
		return PhasePlayDevelopmentCard(p.game), EnterPush
	case 4: // Game Stats
		return newPhaseGameStats(p.game, p.game.PlayerTurn), EnterPush
	case 5: // End Turn
		return p.game.NextTurn(), EnterReset
	default:
		panic("Invalid option selected")
	}
}

func (p *phaseIdle) HelpText() string {
	if p.notification != "" {
		return p.notification + " Anything else?"
	}
	return "What do you want to do?"
}

func saveGameState(g *Game) error {
	filename := fmt.Sprintf("game_save_%s.gob", time.Now().Format("2006-01-02_15-04-05"))
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(g); err != nil {
		return fmt.Errorf("encoding failed: %w", err)
	}
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

func (g *Game) save() error {
	if g.saveHandler == nil {
		return saveGameState(g)
	}
	notice, err := g.saveHandler(g)
	if err == nil {
		g.saveNotice = notice
	}
	return err
}
