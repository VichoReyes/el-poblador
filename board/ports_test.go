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
