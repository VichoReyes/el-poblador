package game

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/VichoReyes/el-poblador/board"
)

func awardGame() *Game {
	g := &Game{}
	g.StartOptions([]string{"Ana", "Bob", "Cat"}, false, true)
	g.SetPhase(PhaseIdle(g))
	return g
}

func roadLine(g *Game, owner, x, y, length int) {
	for i := 0; i < length; i++ {
		g.Board.SetRoad(board.NewPathCoord(board.CrossCoord{X: x, Y: y + i}, board.CrossCoord{X: x, Y: y + i + 1}), owner)
	}
}

func knights(p *Player, count int) {
	p.PlayedDevCards = nil
	for i := 0; i < count; i++ {
		p.PlayedDevCards = append(p.PlayedDevCards, DevCardKnight)
	}
}

func TestAwardThresholdsTiesAndTransfers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		minimum int
	}{
		{"Longest Road", 5}, {"Largest Army", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := awardGame()
			holder := -1
			for _, step := range []struct {
				counts []int
				want   int
			}{
				{[]int{tc.minimum - 1, 0, 0}, -1},
				{[]int{tc.minimum, 0, 0}, 0},
				{[]int{tc.minimum, tc.minimum, 0}, 0},
				{[]int{tc.minimum, tc.minimum + 1, 0}, 1},
				{[]int{tc.minimum + 1, tc.minimum + 1, 0}, 1},
				{[]int{tc.minimum, tc.minimum, tc.minimum}, 1},
				{[]int{tc.minimum, 0, tc.minimum}, -1},
				{[]int{tc.minimum, 0, tc.minimum + 1}, 2},
				{[]int{0, 0, 0}, -1},
			} {
				holder = g.updateAward(tc.name, holder, step.counts, tc.minimum)
				if holder != step.want {
					t.Fatalf("counts %v: holder = %d, want %d", step.counts, holder, step.want)
				}
			}
		})
	}
}

func TestAwardsScoreOnlyPlayedKnightsAndContinuousRoads(t *testing.T) {
	g := awardGame()
	g.Players[0].HiddenDevCards = []DevCard{DevCardKnight, DevCardKnight, DevCardKnight}
	g.Players[1].PlayedDevCards = []DevCard{DevCardMonopoly, DevCardYearOfPlenty, DevCardRoadBuilding}
	roadLine(g, 0, 0, 2, 4)
	g.updateAwards()
	if g.LongestRoadHolder != -1 || g.LargestArmyHolder != -1 {
		t.Fatal("awards claimed below their thresholds")
	}
	roadLine(g, 0, 0, 2, 5)
	knights(&g.Players[0], 3)
	g.updateAwards()
	if g.LongestRoadHolder != 0 || g.LargestArmyHolder != 0 || g.Players[0].VictoryPoints(g) != 4 {
		t.Fatal("both awards should grant player 0 four points")
	}
	roadLine(g, 1, 2, 0, 6)
	knights(&g.Players[1], 4)
	g.updateAwards()
	if g.Players[0].VictoryPoints(g) != 0 || g.Players[1].VictoryPoints(g) != 4 {
		t.Fatal("award points did not transfer to player 1")
	}
}

func TestSettlementReassignsLongestRoad(t *testing.T) {
	for _, tied := range []bool{false, true} {
		t.Run(map[bool]string{false: "transfer", true: "unclaimed tie"}[tied], func(t *testing.T) {
			g := awardGame()
			roadLine(g, 0, 0, 2, 6)
			roadLine(g, 1, 2, 0, 5)
			if tied {
				roadLine(g, 2, 5, 2, 5)
			}
			g.updateAwards()
			if g.LongestRoadHolder != 0 {
				t.Fatal("expected player 0 to hold Longest Road")
			}
			g.PlayerTurn = 2
			cross := board.CrossCoord{X: 0, Y: 4}
			g.Board.SetRoad(board.NewPathCoord(cross, board.CrossCoord{X: 1, Y: 4}), 2)
			for _, resource := range board.RESOURCE_TYPES {
				g.Players[2].Resources[resource] = 1
			}
			g.SetPhase(&phaseSettlementPlacement{game: g, cursorCross: cross})
			g.ConfirmAction(nil)
			want := 1
			if tied {
				want = -1
			}
			if g.LongestRoadHolder != want || g.Players[0].VictoryPoints(g) != 0 {
				t.Fatalf("after settlement, holder = %d, want %d", g.LongestRoadHolder, want)
			}
		})
	}
}

func TestTransferredRoadWinsOnRecipientsTurn(t *testing.T) {
	g := awardGame()
	roadLine(g, 0, 0, 2, 6)
	roadLine(g, 1, 2, 0, 5)
	for i := 0; i < 8; i++ {
		g.Players[1].HiddenDevCards = append(g.Players[1].HiddenDevCards, DevCardVictoryPoint)
	}
	g.updateAwards()
	g.PlayerTurn = 2
	cross := board.CrossCoord{X: 0, Y: 4}
	g.Board.SetRoad(board.NewPathCoord(cross, board.CrossCoord{X: 1, Y: 4}), 2)
	for _, resource := range board.RESOURCE_TYPES {
		g.Players[2].Resources[resource] = 1
	}
	g.SetPhase(&phaseSettlementPlacement{game: g, cursorCross: cross})
	g.ConfirmAction(nil)
	if g.LongestRoadHolder != 1 || g.Players[1].VictoryPoints(g) != 10 {
		t.Fatal("settlement should transfer Longest Road and bring player 1 to ten points")
	}
	if _, ended := g.Phase().(*phaseGameEnd); ended {
		t.Fatal("recipient must wait for their own turn to win")
	}
	g.SetPhase(g.NextTurn()) // player 0
	g.SetPhase(g.NextTurn()) // player 1
	if end, ok := g.Phase().(*phaseGameEnd); !ok || end.winner != &g.Players[1] {
		t.Fatalf("recipient did not win at start of turn: %T", g.Phase())
	}
}

func TestRoadBuildingCanWinWithEitherFreeRoad(t *testing.T) {
	for _, length := range []int{3, 4} {
		t.Run(map[int]string{3: "second road", 4: "first road"}[length], func(t *testing.T) {
			g := awardGame()
			roadLine(g, 0, 0, 2, length)
			for i := 0; i < 8; i++ {
				g.Players[0].HiddenDevCards = append(g.Players[0].HiddenDevCards, DevCardVictoryPoint)
			}
			g.Players[0].HiddenDevCards = append(g.Players[0].HiddenDevCards, DevCardRoadBuilding)
			g.SetPhase(PhasePlayDevelopmentCard(g))
			g.Phase().(*phasePlayDevelopmentCard).selected = 8
			g.ConfirmAction(nil)
			available := g.Players[0].RoadsAvailable
			for i := length; i < 5; i++ {
				start, ok := g.Phase().(*phaseRoadStart)
				if !ok {
					t.Fatalf("expected free road start, got %T", g.Phase())
				}
				start.cursorCross = board.CrossCoord{X: 0, Y: 2 + i}
				g.ConfirmAction(nil)
				g.Phase().(*phaseRoadEnd).cursorCross = board.CrossCoord{X: 0, Y: 3 + i}
				g.ConfirmAction(nil)
			}
			if end, ok := g.Phase().(*phaseGameEnd); !ok || end.winner != &g.Players[0] {
				t.Fatalf("free road did not win: %T", g.Phase())
			}
			if g.Players[0].RoadsAvailable != available-(5-length) {
				t.Fatal("free roads did not consume road pieces")
			}
		})
	}
}

func TestPurchasedRoadCanWin(t *testing.T) {
	g := awardGame()
	roadLine(g, 0, 0, 2, 4)
	for i := 0; i < 8; i++ {
		g.Players[0].HiddenDevCards = append(g.Players[0].HiddenDevCards, DevCardVictoryPoint)
	}
	g.Players[0].Resources[board.ResourceWood] = 1
	g.Players[0].Resources[board.ResourceBrick] = 1
	p := PhaseRoadEnd(g, board.CrossCoord{X: 0, Y: 6}).(*phaseRoadEnd)
	p.cursorCross = board.CrossCoord{X: 0, Y: 7}
	g.SetPhase(p)
	g.ConfirmAction(nil)
	if _, ok := g.Phase().(*phaseGameEnd); !ok || g.LongestRoadHolder != 0 {
		t.Fatalf("purchased road did not win: %T", g.Phase())
	}
}

func TestKnightCanWinBeforeOrAfterRolling(t *testing.T) {
	for _, beforeRoll := range []bool{true, false} {
		t.Run(map[bool]string{true: "before roll", false: "after roll"}[beforeRoll], func(t *testing.T) {
			g := awardGame()
			knights(&g.Players[0], 2)
			g.Players[0].HiddenDevCards = []DevCard{DevCardKnight}
			for i := 0; i < 8; i++ {
				g.Players[0].HiddenDevCards = append(g.Players[0].HiddenDevCards, DevCardVictoryPoint)
			}
			if beforeRoll {
				g.SetPhase(PhaseDiceRoll(g))
				g.Phase().(*phaseDiceRoll).selected = 1
			} else {
				g.SetPhase(PhasePlayDevelopmentCard(g))
			}
			g.ConfirmAction(nil)
			if end, ok := g.Phase().(*phaseGameEnd); !ok || end.winner != &g.Players[0] {
				t.Fatalf("third knight did not win: %T", g.Phase())
			}
			if g.LargestArmyHolder != 0 || g.Players[0].PlayedDevCardCount(DevCardKnight) != 3 {
				t.Fatal("third knight was not counted toward Largest Army")
			}
		})
	}
}

func TestAwardHoldersSurviveSaveAndTie(t *testing.T) {
	g := awardGame()
	roadLine(g, 1, 2, 0, 5)
	knights(&g.Players[1], 3)
	g.updateAwards()
	roadLine(g, 0, 0, 2, 5)
	knights(&g.Players[0], 3)
	g.updateAwards()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(g); err != nil {
		t.Fatal(err)
	}
	var loaded Game
	if err := gob.NewDecoder(&buf).Decode(&loaded); err != nil {
		t.Fatal(err)
	}
	loaded.updateAwards()
	if loaded.LongestRoadHolder != 1 || loaded.LargestArmyHolder != 1 || loaded.Players[1].VictoryPoints(&loaded) != 4 {
		t.Fatal("save/load lost award ownership in a tie")
	}
}
