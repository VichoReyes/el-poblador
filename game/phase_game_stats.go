package game

import "fmt"

// phaseGameStats is a read-only phase opened from either player's idle menu.
type phaseGameStats struct {
	phaseWithOptions
}

func newPhaseGameStats(g *Game, playerID int) *phaseGameStats {
	names := make([]string, len(g.Players))
	for i := range g.Players {
		names[i] = g.Players[i].Name
	}
	return &phaseGameStats{phaseWithOptions: phaseWithOptions{
		game: g, playerID: &playerID, options: names, selected: playerID,
	}}
}

func (p *phaseGameStats) ConfirmTurn() (Phase, EnterMode) {
	return nil, EnterReplace
}

func (p *phaseGameStats) ConfirmSpectator() (SpectatorPhase, EnterMode) {
	return nil, EnterReplace
}

func (p *phaseGameStats) HelpText() string {
	return "Up/Down: select player. Esc: back."
}

func (p *phaseGameStats) Menu() RichText {
	player := &p.game.Players[p.selected]
	menu := RichText{playerText(player, player.Name), {Text: " has played:"}}
	if len(player.PlayedDevCards) == 0 {
		menu = append(menu, TextPart{Text: "\n  No development cards"})
	}
	for _, card := range []DevCard{DevCardKnight, DevCardRoadBuilding, DevCardMonopoly, DevCardYearOfPlenty, DevCardVictoryPoint} {
		if count := player.PlayedDevCardCount(card); count > 0 {
			menu = append(menu, TextPart{Text: fmt.Sprintf("\n  %s: %d", card, count)})
		}
	}
	menu = append(menu, TextPart{Text: "\n"})
	for _, award := range []struct {
		name   string
		holder int
	}{{"Longest Road", p.game.LongestRoadHolder}, {"Largest Army", p.game.LargestArmyHolder}} {
		menu = append(menu, TextPart{Text: "\n" + award.name + ": "})
		if award.holder >= 0 && award.holder < len(p.game.Players) {
			holder := &p.game.Players[award.holder]
			menu = append(menu, playerText(holder, holder.Name), TextPart{Text: " (+2 VP)"})
		} else {
			menu = append(menu, TextPart{Text: "unclaimed"})
		}
	}
	return menu
}
