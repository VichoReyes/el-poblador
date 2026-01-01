package game

import (
	"el_poblador/board"
	"testing"
)

func TestBuildingPhase(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	// Complete initial settlements
	for i := 0; i < 2*len(game.Players); i++ {
		game.MoveCursorToPlaceSettlement()
		game.ConfirmAction(nil) // place settlement
		game.ConfirmAction(nil) // place road
	}
	game.ConfirmAction(nil) // roll dice

	// Handle both possible outcomes: idle phase (normal roll) or robber placement (rolled 7)
	if robberPhase, ok := game.phase.(*phasePlaceRobber); ok {
		// If a 7 was rolled, complete the robber placement to get back to idle
		// Move robber to a different tile
		robberPhase.tileCoord, _ = board.NewTileCoord(1, 2)
		game.ConfirmAction(nil) // place robber
		// This might go to steal phase if there are players adjacent to robber
		// Keep confirming until we get back to idle
		for i := 0; i < 10; i++ { // safety limit
			if _, ok := game.phase.(*phaseIdle); ok {
				break
			}
			game.ConfirmAction(nil)
		}
	}

	// Should be in idle phase now
	if _, ok := game.phase.(*phaseIdle); !ok {
		t.Fatalf("Should be in idle phase after dice roll and robber handling, got: %T", game.phase)
	}

	// Select "Build" option (it's already the first option, so no need to move cursor)
	game.ConfirmAction(nil)

	// Should be in building phase now
	if _, ok := game.phase.(*phaseBuilding); !ok {
		t.Fatal("Should be in building phase after selecting Build")
	}

	// Test cancel functionality
	// Move cursor to "Cancel" (last option, index 4)
	for i := 0; i < 4; i++ {
		game.phase.MoveCursor("down")
	}
	game.ConfirmAction(nil)

	// Should be back in idle phase
	if _, ok := game.phase.(*phaseIdle); !ok {
		t.Fatal("Should be back in idle phase after canceling")
	}
}

func TestRoadPurchase(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	// Directly set up a settlement for the current player to connect roads to
	playerId := game.PlayerTurn
	settlementCoord, _ := board.NewCrossCoord(2, 3)
	game.Board.SetSettlement(settlementCoord, playerId)

	game.phase = PhaseIdle(game)

	// Give the player enough resources to build a road
	player := &game.Players[game.PlayerTurn]
	player.AddResource(board.ResourceWood)
	player.AddResource(board.ResourceBrick)

	// Select "Build" option (first option in idle phase)
	game.ConfirmAction(nil)

	// Should be in building phase now
	if _, ok := game.phase.(*phaseBuilding); !ok {
		t.Fatalf("Should be in building phase after selecting Build, got: %T", game.phase)
	}

	// Select "Road" option (first option in building phase)
	game.ConfirmAction(nil)

	// Should be in road start phase
	roadStartPhase, ok := game.phase.(*phaseRoadStart)
	if !ok {
		t.Fatalf("Should be in road start phase after selecting Road, got: %T", game.phase)
	}

	// Set cursor to the settlement position
	roadStartPhase.cursorCross = settlementCoord

	// Confirm road start position
	game.ConfirmAction(nil)

	// Should be in road end phase
	roadEndPhase, ok := game.phase.(*phaseRoadEnd)
	if !ok {
		t.Fatalf("Should be in road end phase after confirming start, got: %T", game.phase)
	}

	// Find a valid neighbor for the road end
	neighbors := settlementCoord.Neighbors()
	found := false
	for _, neighbor := range neighbors {
		pathCoord := board.NewPathCoord(settlementCoord, neighbor)
		if game.Board.CanPlaceRoad(pathCoord, playerId) {
			roadEndPhase.cursorCross = neighbor
			found = true
			break
		}
	}

	if !found {
		t.Fatal("No valid neighbor found to complete road placement")
	}

	// Confirm road end position to complete the road purchase
	game.ConfirmAction(nil)

	// Verify the road was actually built by checking:
	// 1. Resources were consumed (player should no longer be able to build another road)
	// 2. The path can no longer be placed (because a road is there)
	pathCoord := board.NewPathCoord(settlementCoord, roadEndPhase.cursorCross)
	if player.CanBuildRoad() {
		t.Fatal("Resources were not consumed - player can still build roads")
	}
	if game.Board.CanPlaceRoad(pathCoord, playerId) {
		t.Fatal("Road was not built - path is still available")
	}

	// BUG: This should be idle phase but currently returns to building phase
	if _, ok := game.phase.(*phaseIdle); !ok {
		t.Fatalf("Should be in idle phase after building road, got: %T", game.phase)
	}
}

// TestCityUpgradeFreesSettlement verifies that upgrading a settlement to a city
// should return the settlement piece to the player's available pool
func TestCityUpgradeFreesSettlement(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	player := &game.Players[0]
	settlementCoord, _ := board.NewCrossCoord(2, 3)

	initialSettlements := player.SettlementsAvailable
	initialCities := player.CitiesAvailable

	// Place settlement using the initial settlement phase (doesn't require roads)
	initialPhase := &phaseInitialSettlements{
		game:        game,
		cursorCross: settlementCoord,
		isFirstPair: true,
	}

	initialPhase.Confirm() // This places settlement on the board

	// BUG: Initial settlements should consume from the available pool
	if player.SettlementsAvailable != initialSettlements-1 {
		t.Fatalf("Initial settlement should decrement available settlements from %d to %d, got %d",
			initialSettlements, initialSettlements-1, player.SettlementsAvailable)
	}

	settlementsAfterBuilding := player.SettlementsAvailable

	// Give player resources for city and upgrade using the phase
	player.AddResource(board.ResourceWheat)
	player.AddResource(board.ResourceWheat)
	player.AddResource(board.ResourceOre)
	player.AddResource(board.ResourceOre)
	player.AddResource(board.ResourceOre)

	cityPhase := &phaseCityPlacement{
		game:        game,
		cursorCross: settlementCoord,
	}

	cityPhase.Confirm() // This upgrades to city

	// BUG: After upgrading to city, the settlement should be freed
	// The settlement piece should be returned to the available pool
	if player.SettlementsAvailable != settlementsAfterBuilding+1 {
		t.Fatalf("Expected settlement to be freed after city upgrade (should increase from %d to %d), got %d",
			settlementsAfterBuilding, settlementsAfterBuilding+1, player.SettlementsAvailable)
	}
	if player.CitiesAvailable != initialCities-1 {
		t.Fatalf("Expected city to be used after city upgrade (should decrease from %d to %d), got %d",
			initialCities, initialCities-1, player.CitiesAvailable)
	}
}

// TestBuildSettlementDoesNotConsumeResourcesWhenFailing verifies that when
// BuildSettlement fails, it doesn't consume resources
func TestBuildSettlementDoesNotConsumeResourcesWhenFailing(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	player := &game.Players[0]

	// Set settlements available to 0
	player.SettlementsAvailable = 0

	// Give player resources for a settlement
	player.AddResource(board.ResourceWood)
	player.AddResource(board.ResourceBrick)
	player.AddResource(board.ResourceWheat)
	player.AddResource(board.ResourceSheep)

	resourcesBefore := player.TotalResources()

	// Call BuildSettlement - it should fail
	success := player.BuildSettlement()

	if success {
		t.Fatal("BuildSettlement should have returned false when SettlementsAvailable is 0")
	}

	resourcesAfter := player.TotalResources()

	// BUG: BuildSettlement() consumes resources even when it fails
	if resourcesAfter != resourcesBefore {
		t.Fatalf("BuildSettlement should not consume resources when failing, but consumed %d resources",
			resourcesBefore-resourcesAfter)
	}
}

// TestBuildRoadDoesNotConsumeResourcesWhenFailing verifies that when
// BuildRoad fails, it doesn't consume resources
func TestBuildRoadDoesNotConsumeResourcesWhenFailing(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	player := &game.Players[0]

	// Set roads available to 0
	player.RoadsAvailable = 0

	// Give player resources for a road
	player.AddResource(board.ResourceWood)
	player.AddResource(board.ResourceBrick)

	resourcesBefore := player.TotalResources()

	// Call BuildRoad - it should fail
	success := player.BuildRoad()

	if success {
		t.Fatal("BuildRoad should have returned false when RoadsAvailable is 0")
	}

	resourcesAfter := player.TotalResources()

	// BUG: BuildRoad() consumes resources even when it fails
	if resourcesAfter != resourcesBefore {
		t.Fatalf("BuildRoad should not consume resources when failing, but consumed %d resources",
			resourcesBefore-resourcesAfter)
	}
}

// TestBuildCityDoesNotConsumeResourcesWhenFailing verifies that when
// BuildCity fails, it doesn't consume resources
func TestBuildCityDoesNotConsumeResourcesWhenFailing(t *testing.T) {
	game := &Game{}
	game.Start([]string{"p1", "p2", "p3"})

	player := &game.Players[0]

	// Set cities available to 0
	player.CitiesAvailable = 0

	// Give player resources for a city
	player.AddResource(board.ResourceWheat)
	player.AddResource(board.ResourceWheat)
	player.AddResource(board.ResourceOre)
	player.AddResource(board.ResourceOre)
	player.AddResource(board.ResourceOre)

	resourcesBefore := player.TotalResources()

	// Call BuildCity - it should fail
	success := player.BuildCity()

	if success {
		t.Fatal("BuildCity should have returned false when CitiesAvailable is 0")
	}

	resourcesAfter := player.TotalResources()

	// BUG: BuildCity() consumes resources even when it fails
	if resourcesAfter != resourcesBefore {
		t.Fatalf("BuildCity should not consume resources when failing, but consumed %d resources",
			resourcesBefore-resourcesAfter)
	}
}
