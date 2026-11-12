package game

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/VichoReyes/el-poblador/board"
)

func newPortTradeGame() *Game {
	g := &Game{}
	g.StartOptions([]string{"Alice", "Bob", "Charlie"}, false, true)
	g.PlayerTurn = 1
	g.SetPhase(PhaseIdle(g))
	for i := range g.Players {
		for _, resource := range board.RESOURCE_TYPES {
			g.Players[i].Resources[resource] = 10
		}
	}
	return g
}

func grantPort(t *testing.T, g *Game, playerID int, resource board.ResourceType) {
	t.Helper()
	for path, portResource := range g.Board.Ports {
		if portResource == resource {
			for _, endpoint := range []board.CrossCoord{path.From, path.To} {
				if g.Board.SetSettlement(endpoint, playerID) {
					return
				}
			}
		}
	}
	t.Fatalf("could not settle a %s port", resource)
}

// Use the same perspective routing and cursor actions as local and SSH players.
func selectTradeResources(g *Game, amounts map[board.ResourceType]int, perspective *int) {
	for _, resource := range board.RESOURCE_TYPES {
		for i := 0; i < amounts[resource]; i++ {
			g.MoveCursor("right", perspective)
		}
		g.MoveCursor("down", perspective)
	}
}

func enterTrade(t *testing.T, g *Game, perspective *int) {
	t.Helper()
	if perspective == nil || *perspective == g.PlayerTurn {
		g.MoveCursor("down", perspective) // Make Trade Offer
	} else {
		g.SetSpectatorPhase(*perspective, PhaseSpectatorIdle(g, *perspective))
	}
	g.ConfirmAction(perspective)
	if perspective == nil || *perspective == g.PlayerTurn {
		if _, ok := g.Phase().(*phaseTradeOffer); !ok {
			t.Fatalf("expected trade offer phase, got %T", g.Phase())
		}
	} else if phase, ok := g.SpectatorPhase(*perspective); !ok {
		t.Fatal("missing spectator trade phase")
	} else if _, ok := phase.(*phaseTradeOffer); !ok {
		t.Fatalf("expected spectator trade offer phase, got %T", phase)
	}
}

func proposeTrade(t *testing.T, g *Game, offer, request map[board.ResourceType]int, perspective *int) {
	t.Helper()
	enterTrade(t, g, perspective)
	selectTradeResources(g, offer, perspective)
	g.ConfirmAction(perspective)
	selectTradeResources(g, request, perspective)
	g.ConfirmAction(perspective)
}

func TestPortBankTrades(t *testing.T) {
	for _, resource := range board.RESOURCE_TYPES {
		for _, cost := range []int{2, 3, 4} {
			t.Run(fmt.Sprintf("%s/%d:1", resource, cost), func(t *testing.T) {
				g := newPortTradeGame()
				if cost == 2 {
					grantPort(t, g, g.PlayerTurn, resource)
				} else if cost == 3 {
					grantPort(t, g, g.PlayerTurn, board.ResourceInvalid)
				}
				requested := board.ResourceWheat
				if resource == requested {
					requested = board.ResourceOre
				}
				before := maps.Clone(g.Players[g.PlayerTurn].Resources)
				proposeTrade(t, g, map[board.ResourceType]int{resource: cost}, map[board.ResourceType]int{requested: 1}, nil)
				before[resource] -= cost
				before[requested]++
				if !maps.Equal(g.Players[g.PlayerTurn].Resources, before) {
					t.Fatalf("incorrect resources after trade: %v, want %v", g.Players[g.PlayerTurn].Resources, before)
				}
				if len(g.TradeOffers) != 0 {
					t.Fatal("bank trade created a player offer")
				}
				notification := fmt.Sprintf("Traded %d %s for 1 %s!", cost, resource, requested)
				if _, ok := g.Phase().(*phaseIdle); !ok || !strings.Contains(g.Phase().HelpText(), notification) {
					t.Fatalf("expected idle notification %q, got %T: %s", notification, g.Phase(), g.Phase().HelpText())
				}
				log := fmt.Sprintf("Bob traded %d %s for 1 %s with the bank", cost, resource, requested)
				if got := g.ActionLog[0].String(); got != log {
					t.Fatalf("got log %q, want %q", got, log)
				}
			})
		}
	}
}

func TestPortTradeFallbackAndAvailableRates(t *testing.T) {
	tests := []struct {
		name     string
		ports    []board.ResourceType
		opponent bool
		roadOnly bool
		offer    map[board.ResourceType]int
		request  map[board.ResourceType]int
		bank     bool
	}{
		{name: "no port 3:1", offer: map[board.ResourceType]int{board.ResourceWood: 3}},
		{name: "no port 2:1", offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "generic port cannot trade 2:1", ports: []board.ResourceType{board.ResourceInvalid}, offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "wrong resource port", ports: []board.ResourceType{board.ResourceBrick}, offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "requested resource port", ports: []board.ResourceType{board.ResourceWheat}, offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "specific port does not grant 3:1", ports: []board.ResourceType{board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 3}},
		{name: "opponent port", ports: []board.ResourceType{board.ResourceWood}, opponent: true, offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "road alone", roadOnly: true, offer: map[board.ResourceType]int{board.ResourceWood: 2}},
		{name: "mixed offer", ports: []board.ResourceType{board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 1, board.ResourceBrick: 1}},
		{name: "multiple requested resources", ports: []board.ResourceType{board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 2}, request: map[board.ResourceType]int{board.ResourceWheat: 1, board.ResourceOre: 1}},
		{name: "same resource", ports: []board.ResourceType{board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 2}, request: map[board.ResourceType]int{board.ResourceWood: 1}},
		{name: "4:1 still works with specific port", ports: []board.ResourceType{board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 4}, bank: true},
		{name: "4:1 still works with generic port", ports: []board.ResourceType{board.ResourceInvalid}, offer: map[board.ResourceType]int{board.ResourceWood: 4}, bank: true},
		{name: "generic plus specific gives 3:1", ports: []board.ResourceType{board.ResourceInvalid, board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 3}, bank: true},
		{name: "generic plus specific gives 2:1", ports: []board.ResourceType{board.ResourceInvalid, board.ResourceWood}, offer: map[board.ResourceType]int{board.ResourceWood: 2}, bank: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newPortTradeGame()
			owner := g.PlayerTurn
			if tt.opponent {
				owner = 0
			}
			for _, resource := range tt.ports {
				grantPort(t, g, owner, resource)
			}
			if tt.roadOnly {
				for path := range g.Board.Ports {
					g.Board.SetRoad(path, g.PlayerTurn)
				}
			}
			request := tt.request
			if request == nil {
				request = map[board.ResourceType]int{board.ResourceWheat: 1}
			}
			before := maps.Clone(g.Players[g.PlayerTurn].Resources)
			proposeTrade(t, g, tt.offer, request, nil)
			if tt.bank {
				for resource, amount := range tt.offer {
					before[resource] -= amount
				}
				for resource, amount := range request {
					before[resource] += amount
				}
				if len(g.TradeOffers) != 0 {
					t.Fatal("available bank rate created a player offer")
				}
			} else {
				if len(g.TradeOffers) != 1 {
					t.Fatalf("expected one player offer, got %d", len(g.TradeOffers))
				}
				offer := g.TradeOffers[0]
				if offer.OffererID != g.PlayerTurn || offer.Status != OfferIsPending || offer.TargetID != -1 {
					t.Fatalf("unexpected player offer: %+v", offer)
				}
				for _, resource := range board.RESOURCE_TYPES {
					if offer.Offering[resource] != tt.offer[resource] || offer.Requesting[resource] != request[resource] {
						t.Fatalf("unexpected player offer amounts: %+v", offer)
					}
				}
			}
			if !maps.Equal(g.Players[g.PlayerTurn].Resources, before) {
				t.Fatalf("incorrect resources after trade: %v, want %v", g.Players[g.PlayerTurn].Resources, before)
			}
		})
	}
}

func TestSpectatorsCannotUseBankOrPortTrades(t *testing.T) {
	for _, cost := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("%d:1", cost), func(t *testing.T) {
			g := newPortTradeGame()
			spectator := 0
			grantPort(t, g, spectator, board.ResourceWood)
			grantPort(t, g, spectator, board.ResourceInvalid)
			before := maps.Clone(g.Players[spectator].Resources)
			proposeTrade(t, g, map[board.ResourceType]int{board.ResourceWood: cost}, map[board.ResourceType]int{board.ResourceWheat: 1}, &spectator)
			if !maps.Equal(g.Players[spectator].Resources, before) {
				t.Fatal("spectator executed a bank trade")
			}
			if len(g.TradeOffers) != 1 || g.TradeOffers[0].OffererID != spectator {
				t.Fatalf("expected spectator player offer, got %v", g.TradeOffers)
			}
			phase, _ := g.SpectatorPhase(spectator)
			if _, ok := phase.(*phaseSpectatorIdle); !ok {
				t.Fatalf("expected spectator idle phase, got %T", phase)
			}
		})
	}
}

func TestPortTradeRechecksResources(t *testing.T) {
	for _, cost := range []int{2, 3} {
		t.Run(fmt.Sprintf("%d:1", cost), func(t *testing.T) {
			g := newPortTradeGame()
			grantPort(t, g, g.PlayerTurn, board.ResourceWood)
			grantPort(t, g, g.PlayerTurn, board.ResourceInvalid)
			enterTrade(t, g, nil)
			selectTradeResources(g, map[board.ResourceType]int{board.ResourceWood: cost}, nil)
			g.ConfirmAction(nil)
			selectTradeResources(g, map[board.ResourceType]int{board.ResourceWheat: 1}, nil)
			// Resources may have changed since the offer was selected.
			g.Players[g.PlayerTurn].Resources[board.ResourceWood] = cost - 1
			before := maps.Clone(g.Players[g.PlayerTurn].Resources)
			g.ConfirmAction(nil)
			if !maps.Equal(g.Players[g.PlayerTurn].Resources, before) || len(g.TradeOffers) != 0 || len(g.ActionLog) != 0 {
				t.Fatal("unaffordable bank trade changed state")
			}
			if !strings.Contains(g.Phase().HelpText(), "Not enough resources for bank trade!") {
				t.Fatalf("missing insufficient resources notification: %s", g.Phase().HelpText())
			}
		})
	}
}

func TestPortTradeAfterSaveLoad(t *testing.T) {
	g := newPortTradeGame()
	grantPort(t, g, g.PlayerTurn, board.ResourceWood)
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(g); err != nil {
		t.Fatal(err)
	}
	var loaded Game
	if err := gob.NewDecoder(&buf).Decode(&loaded); err != nil {
		t.Fatal(err)
	}
	loaded.SetPhase(PhaseIdle(&loaded))
	if !maps.Equal(loaded.Board.Ports, g.Board.Ports) {
		t.Fatal("save/load changed ports")
	}
	proposeTrade(t, &loaded, map[board.ResourceType]int{board.ResourceWood: 2}, map[board.ResourceType]int{board.ResourceWheat: 1}, nil)
	if loaded.Players[loaded.PlayerTurn].Resources[board.ResourceWood] != 8 || loaded.Players[loaded.PlayerTurn].Resources[board.ResourceWheat] != 11 || len(loaded.TradeOffers) != 0 {
		t.Fatal("port access did not survive save/load")
	}
}

func TestPortTradeAfterBuildingAndUpgrading(t *testing.T) {
	g := newPortTradeGame()
	var cross board.CrossCoord
	for path, resource := range g.Board.Ports {
		if resource == board.ResourceWood {
			cross = path.From
			break
		}
	}
	g.Board.SetRoad(board.NewPathCoord(cross, cross.Neighbors()[0]), g.PlayerTurn)
	menu := PhaseTradeOffer(g)
	if !strings.Contains(menu.Menu().String(), "Wood:  0 / 10 (bank 4:1)") {
		t.Fatal("road alone changed the bank rate")
	}
	settlement := PhaseSettlementPlacement(g).(*phaseSettlementPlacement)
	settlement.cursorCross = cross
	g.SetPhase(settlement)
	g.ConfirmAction(nil)
	if !g.Board.HasSettlementAt(cross, g.PlayerTurn) {
		t.Fatal("failed to build port settlement")
	}
	if !strings.Contains(menu.Menu().String(), "Wood:  0 / 9 (bank 2:1)") {
		t.Fatal("trade menu did not pick up newly built port")
	}
	proposeTrade(t, g, map[board.ResourceType]int{board.ResourceWood: 2}, map[board.ResourceType]int{board.ResourceWheat: 1}, nil)
	if g.Players[g.PlayerTurn].Resources[board.ResourceWood] != 7 || len(g.TradeOffers) != 0 {
		t.Fatal("new port did not grant an immediate discount")
	}
	city := PhaseCityPlacement(g).(*phaseCityPlacement)
	city.cursorCross = cross
	g.SetPhase(city)
	g.ConfirmAction(nil)
	if _, ok := g.Board.CityUpgrades[cross]; !ok {
		t.Fatal("failed to upgrade port settlement")
	}
	proposeTrade(t, g, map[board.ResourceType]int{board.ResourceWood: 2}, map[board.ResourceType]int{board.ResourceWheat: 1}, nil)
	if g.Players[g.PlayerTurn].Resources[board.ResourceWood] != 5 || len(g.TradeOffers) != 0 {
		t.Fatal("city upgrade lost port access")
	}
	g.SetPhase(g.NextTurn())
	if !strings.Contains(PhaseTradeOffer(g).Menu().String(), "Wood:  0 / 10 (bank 4:1)") {
		t.Fatal("port discount leaked to next turn holder")
	}
}

func TestPortTradeCancellation(t *testing.T) {
	g := newPortTradeGame()
	grantPort(t, g, g.PlayerTurn, board.ResourceWood)
	before := maps.Clone(g.Players[g.PlayerTurn].Resources)
	enterTrade(t, g, nil)
	selectTradeResources(g, map[board.ResourceType]int{board.ResourceWood: 2}, nil)
	g.ConfirmAction(nil)
	selectTradeResources(g, map[board.ResourceType]int{board.ResourceWheat: 1}, nil)
	g.CancelAction(nil)
	if _, ok := g.Phase().(*phaseTradeOffer); !ok {
		t.Fatalf("cancel should return to offer, got %T", g.Phase())
	}
	g.CancelAction(nil)
	if _, ok := g.Phase().(*phaseIdle); !ok {
		t.Fatalf("cancel should return to idle, got %T", g.Phase())
	}
	if !maps.Equal(g.Players[g.PlayerTurn].Resources, before) || len(g.TradeOffers) != 0 || len(g.ActionLog) != 0 {
		t.Fatal("cancelled port trade changed state")
	}
}

func TestPortTradeMenuRates(t *testing.T) {
	g := newPortTradeGame()
	grantPort(t, g, g.PlayerTurn, board.ResourceInvalid)
	grantPort(t, g, g.PlayerTurn, board.ResourceWood)
	grantPort(t, g, 0, board.ResourceBrick)
	menu := PhaseTradeOffer(g).Menu().String()
	for _, resource := range board.RESOURCE_TYPES {
		rate := 3
		if resource == board.ResourceWood {
			rate = 2
		}
		want := fmt.Sprintf("%s:  0 / 10 (bank %d:1)", resource, rate)
		if !strings.Contains(menu, want) {
			t.Fatalf("menu missing %q: %s", want, menu)
		}
	}
	if strings.Contains(menu, "\x1b[") {
		t.Fatal("trade menu stored ANSI styles")
	}
	if menu := PhaseTradeOfferForPlayer(g, 0).Menu().String(); strings.Contains(menu, "bank") {
		t.Fatal("spectator menu advertised bank trades")
	}
}
