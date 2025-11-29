package game

import (
	"el_poblador/board"
	"testing"
)

// TestTurnHolderAndSpectatorTrading tests that both turn holders and spectators
// can create and accept trade offers
func TestTurnHolderAndSpectatorTrading(t *testing.T) {
	// Setup game
	g := &Game{}
	g.Start([]string{"Alice", "Bob", "Charlie"})

	// Give players some resources
	g.Players[0].Resources[board.ResourceWood] = 5
	g.Players[0].Resources[board.ResourceBrick] = 0

	g.Players[1].Resources[board.ResourceWood] = 0
	g.Players[1].Resources[board.ResourceBrick] = 5

	// Set turn to player 0 (Alice)
	g.PlayerTurn = 0
	g.phase = PhaseIdle(g)

	// Initialize spectator phases (simulate dice roll)
	g.spectatorPhases = make(map[int]SpectatorPhase)
	for i := range g.Players {
		if i != g.PlayerTurn {
			g.spectatorPhases[i] = PhaseSpectatorIdle(g, i)
		}
	}

	// Turn holder (Alice) creates a trade offer: 2 wood for 2 brick
	offerPhase := PhaseTradeOffer(g).(*phaseTradeOffer)
	offerPhase.offer[board.ResourceWood] = 2

	receivePhase := offerPhase.Confirm().(*phaseTradeSelectReceive)
	receivePhase.request[board.ResourceBrick] = 2

	// Confirm the trade offer - this should create it in the pool
	receivePhase.Confirm()

	// Verify the offer was created
	if len(g.TradeOffers) != 1 {
		t.Fatalf("Expected 1 trade offer, got %d", len(g.TradeOffers))
	}

	offer := &g.TradeOffers[0]
	if offer.OffererID != 0 {
		t.Errorf("Expected offerer to be player 0, got %d", offer.OffererID)
	}
	if offer.Offering[board.ResourceWood] != 2 {
		t.Errorf("Expected offering 2 wood, got %d", offer.Offering[board.ResourceWood])
	}
	if offer.Requesting[board.ResourceBrick] != 2 {
		t.Errorf("Expected requesting 2 brick, got %d", offer.Requesting[board.ResourceBrick])
	}

	// Spectator (Bob, player 1) views and accepts the offer
	spectatorPhase := g.spectatorPhases[1]

	// Navigate to "View Trade Pool"
	spectatorIdle := spectatorPhase.(*phaseSpectatorIdle)
	spectatorIdle.MoveCursor("down") // Move to "View Trade Pool"

	// Confirm to enter trade pool
	tradePoolPhase := spectatorIdle.Confirm(g).(*phaseSpectatorTradePool)

	// Verify we can see the offer
	if len(tradePoolPhase.offersSnapshot) != 1 {
		t.Fatalf("Expected to see 1 offer in pool, got %d", len(tradePoolPhase.offersSnapshot))
	}

	// Check if Bob can take the offer
	canTake := offer.canTake(1, g)
	if canTake != CanTakeTrue {
		t.Fatalf("Expected Bob to be able to take the offer, got status %d", canTake)
	}

	// Record resources before trade
	aliceWoodBefore := g.Players[0].Resources[board.ResourceWood]
	aliceBrickBefore := g.Players[0].Resources[board.ResourceBrick]
	bobWoodBefore := g.Players[1].Resources[board.ResourceWood]
	bobBrickBefore := g.Players[1].Resources[board.ResourceBrick]

	// Accept the offer (it's already selected at index 0)
	tradePoolPhase.Confirm(g)

	// Verify resources were transferred
	aliceWoodAfter := g.Players[0].Resources[board.ResourceWood]
	aliceBrickAfter := g.Players[0].Resources[board.ResourceBrick]
	bobWoodAfter := g.Players[1].Resources[board.ResourceWood]
	bobBrickAfter := g.Players[1].Resources[board.ResourceBrick]

	// Alice should have lost 2 wood and gained 2 brick
	if aliceWoodAfter != aliceWoodBefore-2 {
		t.Errorf("Alice should have lost 2 wood, had %d, now has %d", aliceWoodBefore, aliceWoodAfter)
	}
	if aliceBrickAfter != aliceBrickBefore+2 {
		t.Errorf("Alice should have gained 2 brick, had %d, now has %d", aliceBrickBefore, aliceBrickAfter)
	}

	// Bob should have gained 2 wood and lost 2 brick
	if bobWoodAfter != bobWoodBefore+2 {
		t.Errorf("Bob should have gained 2 wood, had %d, now has %d", bobWoodBefore, bobWoodAfter)
	}
	if bobBrickAfter != bobBrickBefore-2 {
		t.Errorf("Bob should have lost 2 brick, had %d, now has %d", bobBrickBefore, bobBrickAfter)
	}

	// Verify the offer status changed to completed
	if offer.Status != OfferIsCompleted {
		t.Errorf("Expected offer status to be completed, got %d", offer.Status)
	}
}

// TestTradePoolStaysOpenOnFailedAccept verifies that when a player tries to accept
// a trade they cannot take, they remain in the trade pool phase
func TestTradePoolStaysOpenOnFailedAccept(t *testing.T) {
	// Setup game
	g := &Game{}
	g.Start([]string{"Alice", "Bob", "Charlie"})

	// Give Alice resources to make an offer
	g.Players[0].Resources[board.ResourceWood] = 5

	// Bob has NO brick - he won't be able to accept the trade
	g.Players[1].Resources[board.ResourceBrick] = 0

	// Set turn to player 0 (Alice)
	g.PlayerTurn = 0
	g.phase = PhaseIdle(g)

	// Initialize spectator phases
	g.spectatorPhases = make(map[int]SpectatorPhase)
	for i := range g.Players {
		if i != g.PlayerTurn {
			g.spectatorPhases[i] = PhaseSpectatorIdle(g, i)
		}
	}

	// Alice creates a trade offer: 2 wood for 2 brick
	success := g.AddTradeOffer(0, -1, map[board.ResourceType]int{
		board.ResourceWood: 2,
	}, map[board.ResourceType]int{
		board.ResourceBrick: 2,
	})

	if !success {
		t.Fatal("Failed to create trade offer")
	}

	// Bob tries to accept the offer but doesn't have the resources
	spectatorPhase := g.spectatorPhases[1]
	spectatorIdle := spectatorPhase.(*phaseSpectatorIdle)
	spectatorIdle.MoveCursor("down") // Move to "View Trade Pool"

	// Enter trade pool
	tradePoolPhase := spectatorIdle.Confirm(g).(*phaseSpectatorTradePool)

	// Try to confirm the offer (Bob doesn't have 2 brick)
	resultPhase := tradePoolPhase.Confirm(g)

	// Verify Bob stays in the trade pool phase (not returned to idle)
	if _, ok := resultPhase.(*phaseSpectatorTradePool); !ok {
		t.Errorf("Expected to stay in trade pool phase, got %T", resultPhase)
	}

	// Verify the trade was NOT executed (resources unchanged)
	if g.Players[0].Resources[board.ResourceWood] != 5 {
		t.Errorf("Alice's wood should be unchanged at 5, got %d", g.Players[0].Resources[board.ResourceWood])
	}
	if g.Players[1].Resources[board.ResourceBrick] != 0 {
		t.Errorf("Bob's brick should be unchanged at 0, got %d", g.Players[1].Resources[board.ResourceBrick])
	}

	// Verify the offer is still pending
	if g.TradeOffers[0].Status != OfferIsPending {
		t.Errorf("Offer should still be pending, got status %d", g.TradeOffers[0].Status)
	}
}
