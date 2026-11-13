package board

// LongestRoad returns the longest continuous trail of a player's roads. A trail
// may revisit a crossing, but never a road, and cannot pass through an opponent's
// settlement (cities remain in Settlements as well).
func (b *Board) LongestRoad(playerID int) int {
	adjacent := make(map[CrossCoord][]PathCoord)
	for road, owner := range b.Roads {
		if owner == playerID {
			adjacent[road.From] = append(adjacent[road.From], road)
			adjacent[road.To] = append(adjacent[road.To], road)
		}
	}
	used := make(map[PathCoord]bool)
	var walk func(CrossCoord, int) int
	walk = func(cross CrossCoord, length int) int {
		if owner, occupied := b.Settlements[cross]; length > 0 && occupied && owner != playerID {
			return length
		}
		longest := length
		for _, road := range adjacent[cross] {
			if used[road] {
				continue
			}
			next := road.To
			if next == cross {
				next = road.From
			}
			used[road] = true
			longest = max(longest, walk(next, length+1))
			delete(used, road)
		}
		return longest
	}
	longest := 0
	for cross := range adjacent {
		longest = max(longest, walk(cross, 0))
	}
	return longest
}
