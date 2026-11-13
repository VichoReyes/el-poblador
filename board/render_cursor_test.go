package board

import (
	"strings"
	"testing"
)

func TestCrossCursorPreservesBuilding(t *testing.T) {
	cross := CrossCoord{X: 2, Y: 4}
	b := NewDesertBoard()
	b.Settlements[cross] = 0
	view := BoardView{CrossCursor: &cross}
	if !strings.Contains(strings.Join(b.Print(view), "\n"), "▲▲▲") {
		t.Fatal("cursor hid settlement")
	}
	b.CityUpgrades[cross] = 0
	if !strings.Contains(strings.Join(b.Print(view), "\n"), "███") {
		t.Fatal("cursor hid city")
	}
}
