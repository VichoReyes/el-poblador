package board

import "testing"

func addRoadTrail(b *Board, owner int, crosses ...CrossCoord) {
	for i := 1; i < len(crosses); i++ {
		b.SetRoad(NewPathCoord(crosses[i-1], crosses[i]), owner)
	}
}

func TestLongestRoad(t *testing.T) {
	line := []CrossCoord{{0, 2}, {0, 3}, {0, 4}, {0, 5}, {0, 6}, {0, 7}, {0, 8}}
	loop := []CrossCoord{{2, 3}, {2, 2}, {3, 2}, {3, 3}, {3, 4}, {2, 4}, {2, 3}}
	for _, tc := range []struct {
		name   string
		trails [][]CrossCoord
		block  *CrossCoord
		owner  int
		want   int
	}{
		{name: "empty"},
		{name: "line", trails: [][]CrossCoord{line}, want: 6},
		{name: "disconnected", trails: [][]CrossCoord{line[:4], line[4:]}, want: 3},
		{name: "fork", trails: [][]CrossCoord{
			{{2, 4}, {2, 3}, {2, 2}, {3, 2}},
			{{2, 4}, {2, 5}, {2, 6}},
			{{2, 4}, {3, 4}, {3, 5}, {4, 5}},
		}, want: 6},
		{name: "loop", trails: [][]CrossCoord{loop}, want: 6},
		{name: "loop with tail", trails: [][]CrossCoord{loop, {{2, 4}, {2, 5}, {2, 6}}}, want: 8},
		{name: "two joined loops", trails: [][]CrossCoord{loop, {{2, 4}, {2, 5}, {2, 6}, {3, 6}, {3, 5}, {3, 4}}}, want: 11},
		{name: "opponent splits road", trails: [][]CrossCoord{line}, block: &CrossCoord{0, 5}, owner: 1, want: 3},
		{name: "own settlement does not split road", trails: [][]CrossCoord{line}, block: &CrossCoord{0, 5}, owner: 0, want: 6},
		{name: "opponent at endpoint", trails: [][]CrossCoord{line}, block: &CrossCoord{0, 2}, owner: 1, want: 6},
		{name: "opponent opens loop", trails: [][]CrossCoord{loop}, block: &CrossCoord{2, 3}, owner: 1, want: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewDesertBoard()
			for _, trail := range tc.trails {
				addRoadTrail(b, 0, trail...)
			}
			if tc.block != nil {
				b.Settlements[*tc.block] = tc.owner
				b.CityUpgrades[*tc.block] = tc.owner
			}
			if got := b.LongestRoad(0); got != tc.want {
				t.Fatalf("LongestRoad(0) = %d, want %d", got, tc.want)
			}
			if got := b.LongestRoad(1); got != 0 {
				t.Fatalf("opponent with no roads has length %d", got)
			}
		})
	}
}

func TestOpponentRoadDoesNotConnectTrails(t *testing.T) {
	b := NewDesertBoard()
	addRoadTrail(b, 0, CrossCoord{0, 2}, CrossCoord{0, 3}, CrossCoord{0, 4})
	addRoadTrail(b, 1, CrossCoord{0, 4}, CrossCoord{0, 5})
	addRoadTrail(b, 0, CrossCoord{0, 5}, CrossCoord{0, 6}, CrossCoord{0, 7})
	if got := b.LongestRoad(0); got != 2 {
		t.Fatalf("opponent road joined trails: got %d", got)
	}
}

func TestCannotExtendThroughOpponentSettlement(t *testing.T) {
	b := NewDesertBoard()
	addRoadTrail(b, 0, CrossCoord{2, 2}, CrossCoord{2, 3})
	b.Settlements[CrossCoord{2, 3}] = 1
	road := NewPathCoord(CrossCoord{2, 3}, CrossCoord{2, 4})
	if b.CanPlaceRoad(road, 0) {
		t.Fatal("road may not connect through an opponent's settlement")
	}
	// Reaching the opponent from an independent road at the other endpoint is legal.
	addRoadTrail(b, 0, CrossCoord{2, 4}, CrossCoord{2, 5})
	if !b.CanPlaceRoad(road, 0) {
		t.Fatal("road may end at an opponent's settlement")
	}
}
