package game

// updateAwards runs after roads, settlements, or played knights change, while
// the game is locked for an action. Holders are saved because ties depend on who
// held the award before the change.
func (g *Game) updateAwards() {
	roads := make([]int, len(g.Players))
	armies := make([]int, len(g.Players))
	for i := range g.Players {
		roads[i] = g.Board.LongestRoad(i)
		armies[i] = g.Players[i].PlayedDevCardCount(DevCardKnight)
	}
	g.LongestRoadHolder = g.updateAward("Longest Road", g.LongestRoadHolder, roads, 5)
	g.LargestArmyHolder = g.updateAward("Largest Army", g.LargestArmyHolder, armies, 3)
}

func (g *Game) updateAward(name string, holder int, counts []int, minimum int) int {
	longest := minimum - 1
	next, tied := -1, false
	for playerID, count := range counts {
		if count > longest {
			longest, next, tied = count, playerID, false
		} else if count == longest {
			tied = true
		}
	}
	if holder >= 0 && holder < len(counts) && counts[holder] >= minimum && counts[holder] == longest {
		next = holder
	} else if tied {
		next = -1
	}
	if next != holder {
		if next >= 0 {
			g.LogAction(&g.Players[next], " took ", name, " (+2 victory points)")
		} else if holder >= 0 && holder < len(g.Players) {
			g.LogAction(&g.Players[holder], " lost ", name)
		}
	}
	return next
}
