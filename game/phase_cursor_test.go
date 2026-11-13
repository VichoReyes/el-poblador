package game

import (
	"slices"
	"testing"

	"github.com/VichoReyes/el-poblador/board"
)

func TestCrossCursorMovesAcrossTile(t *testing.T) {
	start := board.CrossCoord{X: 2, Y: 4}
	for _, tc := range []struct {
		direction string
		want      board.CrossCoord
	}{
		{"up", board.CrossCoord{X: 2, Y: 3}}, {"down", board.CrossCoord{X: 2, Y: 5}},
		{"left", board.CrossCoord{X: 1, Y: 4}}, {"right", board.CrossCoord{X: 3, Y: 4}},
	} {
		got, ok := moveCrossCursor(start, tc.direction)
		if !ok || got != tc.want {
			t.Errorf("%s: got %v, %v; want %v", tc.direction, got, ok, tc.want)
		}
	}
	if _, ok := moveCrossCursor(board.CrossCoord{X: 0, Y: 4}, "left"); ok {
		t.Error("cross cursor moved outside board")
	}
}

func TestTileCursorHorizontalLane(t *testing.T) {
	for _, direction := range []string{"left", "right"} {
		start := board.TileCoord{X: 2, Y: 3}
		cursor := start
		for i := 0; i < 2; i++ {
			next, ok := moveTileCursor(cursor, direction)
			if !ok {
				t.Fatalf("%s step %d failed", direction, i)
			}
			cursor = next
		}
		if cursor.Y != start.Y {
			t.Errorf("%s drifted from row %d to %d", direction, start.Y, cursor.Y)
		}
		reverse := "left"
		if direction == "left" {
			reverse = "right"
		}
		for i := 0; i < 2; i++ {
			cursor, _ = moveTileCursor(cursor, reverse)
		}
		if cursor != start {
			t.Errorf("%s and back: got %v, want %v", direction, cursor, start)
		}
	}
}

func TestRoadEndpointsRemainNeighbors(t *testing.T) {
	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			start, ok := board.NewCrossCoord(x, y)
			if !ok {
				continue
			}
			for _, dir := range []string{"up", "down", "left", "right"} {
				initial := &phaseInitialRoad{sourceCross: start, cursorCross: start.Neighbors()[0]}
				road := &phaseRoadEnd{startCross: start, cursorCross: start.Neighbors()[0]}
				initial.MoveCursor(dir)
				road.MoveCursor(dir)
				if !slices.Contains(start.Neighbors(), initial.cursorCross) || !slices.Contains(start.Neighbors(), road.cursorCross) {
					t.Fatalf("invalid road endpoint from %v moving %s", start, dir)
				}
			}
		}
	}
}

func TestTileCursorUsesAvailableHorizontalNeighbor(t *testing.T) {
	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			start, valid := board.NewTileCoord(x, y)
			if !valid {
				continue
			}
			for _, tc := range []struct {
				direction string
				dx        int
			}{{"left", -1}, {"right", 1}} {
				_, above := board.NewTileCoord(x+tc.dx, y-1)
				_, below := board.NewTileCoord(x+tc.dx, y+1)
				got, ok := moveTileCursor(start, tc.direction)
				if ok != (above || below) {
					t.Errorf("from %v %s: got %v, %v; neighbor exists: %v", start, tc.direction, got, ok, above || below)
				}
				if ok && (got.X != x+tc.dx || (got.Y != y-1 && got.Y != y+1)) {
					t.Errorf("from %v %s: invalid destination %v", start, tc.direction, got)
				}
			}
		}
	}
}
