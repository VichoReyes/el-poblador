package game

import "fmt"

type phaseGameEnd struct {
	game   *Game
	winner *Player
}

func PhaseGameEnd(game *Game, winner *Player) Phase {
	return &phaseGameEnd{
		game:   game,
		winner: winner,
	}
}

func (p *phaseGameEnd) ConfirmTurn() (Phase, EnterMode) {
	// Game is over, stay in this phase
	return p, EnterReplace
}

func (p *phaseGameEnd) MoveCursor(direction string) {
	// No cursor movement in game end phase
}

func (p *phaseGameEnd) BoardCursor() interface{} {
	return nil
}

func (p *phaseGameEnd) Menu() RichText {
	menu := RichText{playerText(p.winner, p.winner.Name), {Text: " WINS!\n\nFinal Scores:"}}
	for i := range p.game.Players {
		player := &p.game.Players[i]
		points := player.VictoryPoints(p.game)
		marker := "  "
		if player == p.winner {
			marker = "🏆"
		}
		menu = append(menu, TextPart{Text: "\n" + marker + " "}, playerText(player, player.Name), TextPart{Text: fmt.Sprintf(": %d points", points)})
	}
	return menu
}

func (p *phaseGameEnd) HelpText() string {
	return fmt.Sprintf("Game complete! %s reached 10 victory points!", p.winner.Name)
}
