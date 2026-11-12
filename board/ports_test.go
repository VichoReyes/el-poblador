package board

import (
	"slices"
	"testing"
)

func TestStandardPortLocations(t *testing.T) {
	b := NewLegalBoard(nil)
	if len(b.Ports) != 9 {
		t.Fatalf("got %d ports, want 9", len(b.Ports))
	}
	counts := make(map[ResourceType]int)
	endpoints := make(map[CrossCoord]bool)
	for path, resource := range b.Ports {
		if !slices.Contains(path.From.Neighbors(), path.To) {
			t.Fatalf("port %v does not occupy a path", path)
		}
		if len(coastalTiles(path)) != 1 {
			t.Fatalf("port %v is not coastal", path)
		}
		for _, cross := range []CrossCoord{path.From, path.To} {
			if endpoints[cross] {
				t.Fatalf("ports share endpoint %v", cross)
			}
			endpoints[cross] = true
		}
		counts[resource]++
	}
	if counts[ResourceInvalid] != 4 {
		t.Errorf("got %d generic ports, want 4", counts[ResourceInvalid])
	}
	for _, resource := range RESOURCE_TYPES {
		if counts[resource] != 1 {
			t.Errorf("got %d %s ports, want 1", counts[resource], resource)
		}
	}
}

func coastalTiles(path PathCoord) []TileCoord {
	var tiles []TileCoord
	for _, tile := range path.From.adjacentTileCoords() {
		if slices.Contains(path.To.adjacentTileCoords(), tile) {
			tiles = append(tiles, tile)
		}
	}
	return tiles
}

func TestHasPort(t *testing.T) {
	path := NewPathCoord(CrossCoord{0, 2}, CrossCoord{1, 2})
	tests := []struct {
		name       string
		settlement CrossCoord
		owner      int
		city, road bool
		want       bool
	}{
		{name: "unclaimed"},
		{name: "first endpoint", settlement: path.From, want: true},
		{name: "second endpoint", settlement: path.To, want: true},
		{name: "opponent", settlement: path.From, owner: 1},
		{name: "city", settlement: path.To, city: true, want: true},
		{name: "opponent city", settlement: path.To, owner: 1, city: true},
		{name: "road only", road: true},
		{name: "nearby settlement", settlement: CrossCoord{0, 3}},
	}
	for _, resource := range []ResourceType{ResourceInvalid, ResourceWood} {
		for _, tt := range tests {
			t.Run(resource.String()+"/"+tt.name, func(t *testing.T) {
				b := NewDesertBoard()
				b.Ports = map[PathCoord]ResourceType{path: resource}
				if tt.settlement.IsInBounds() {
					if !b.SetSettlement(tt.settlement, tt.owner) {
						t.Fatal("could not place test settlement")
					}
					if tt.city && !b.UpgradeToCity(tt.settlement, tt.owner) {
						t.Fatal("could not upgrade test city")
					}
				}
				if tt.road {
					b.SetRoad(path, 0)
				}
				if got := b.HasPort(0, resource); got != tt.want {
					t.Errorf("HasPort = %t, want %t", got, tt.want)
				}
				if b.HasPort(0, ResourceBrick) {
					t.Fatal("port granted access to a different port type")
				}
			})
		}
	}
}
