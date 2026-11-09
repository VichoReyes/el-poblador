package game

import (
	"testing"
)

// TestPhaseTradePoolEmptyOffers tests that PhaseTradePool handles the case
// when there are no trade offers without panicking.
// This is a regression test for a bug where empty trade offers would cause
// a panic when calling HelpText() or Menu() methods.
func TestPhaseTradePoolEmptyOffers(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	// Ensure there are no trade offers
	if len(game.TradeOffers) != 0 {
		t.Fatalf("Expected no trade offers, got %d", len(game.TradeOffers))
	}

	// Create a trade pool phase with no offers
	phase := PhaseTradePool(game)

	// These should not panic even when there are no offers
	// On the buggy version, HelpText() will panic due to out-of-bounds access
	// because it calls selectedOffer() which accesses offersSnapshot[p.selected]
	helpText := phase.HelpText()
	if helpText == "" {
		t.Error("HelpText should return a non-empty string")
	}

	menu := phase.Menu().String()
	if menu == "" {
		t.Error("Menu should return a non-empty string")
	}
}
