package board

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/charmbracelet/lipgloss"
)

// NewDesertBoard creates a new board of only desert tiles
func NewDesertBoard() *Board {
	board := &Board{
		Tiles:        make(map[TileCoord]Tile),
		Roads:        make(map[PathCoord]int),
		Settlements:  make(map[CrossCoord]int),
		CityUpgrades: make(map[CrossCoord]int),
		PlayerColors: make(map[int]lipgloss.AdaptiveColor),
	}
	// brute force all tile coords
	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			coord, valid := NewTileCoord(x, y)
			if valid {
				board.Tiles[coord] = Tile{Terrain: TerrainDesert, DiceNumber: 0}
			}
		}
	}
	return board
}

// NewChaoticBoard creates a new board with random tiles
func NewChaoticBoard() *Board {
	board := &Board{
		Tiles:        make(map[TileCoord]Tile),
		Roads:        make(map[PathCoord]int),
		Settlements:  make(map[CrossCoord]int),
		CityUpgrades: make(map[CrossCoord]int),
		PlayerColors: make(map[int]lipgloss.AdaptiveColor),
	}
	// brute force all tile coords
	for x := 0; x <= 5; x++ {
		for y := 0; y <= 10; y++ {
			crossCoord, valid := NewCrossCoord(x, y)
			if valid && rand.IntN(4) == 0 {
				playerId := rand.IntN(4)
				board.Settlements[crossCoord] = playerId
				neighbors := crossCoord.Neighbors()
				board.Roads[NewPathCoord(crossCoord, neighbors[rand.IntN(len(neighbors))])] = playerId
			}
			tileCoord, valid := NewTileCoord(x, y)
			if valid {
				terrain := TerrainType(rand.IntN(6))
				dice := rand.IntN(11) + 2
				if terrain == TerrainDesert {
					dice = 0
				}
				board.Tiles[tileCoord] = Tile{Terrain: terrain, DiceNumber: dice}
			}
		}
	}
	return board
}

var orderedNumbers = []int{5, 2, 6, 3, 8, 10, 9, 12, 11, 4, 8, 10, 9, 4, 5, 6, 3, 11}

// in the order in which catan numbers are filled: from the outside in
// TODO: make it so it can start in any corner
var orderedTiles = []TileCoord{
	{X: 0, Y: 3},
	{X: 0, Y: 5},
	{X: 0, Y: 7},
	{X: 1, Y: 8},
	{X: 2, Y: 9},
	{X: 3, Y: 8},
	{X: 4, Y: 7},
	{X: 4, Y: 5},
	{X: 4, Y: 3},
	{X: 3, Y: 2},
	{X: 2, Y: 1},
	{X: 1, Y: 2},
	{X: 1, Y: 4},
	{X: 1, Y: 6},
	{X: 2, Y: 7},
	{X: 3, Y: 6},
	{X: 3, Y: 4},
	{X: 2, Y: 3},
	{X: 2, Y: 5},
}

func NewLegalBoard(playerColors map[int]lipgloss.AdaptiveColor) *Board {
	diceNumbers := slices.Clone(orderedNumbers)
	terrains := []TerrainType{
		TerrainWood, TerrainWood, TerrainWood, TerrainWood,
		TerrainBrick, TerrainBrick, TerrainBrick,
		TerrainOre, TerrainOre, TerrainOre,
		TerrainWheat, TerrainWheat, TerrainWheat, TerrainWheat,
		TerrainSheep, TerrainSheep, TerrainSheep, TerrainSheep,
		TerrainDesert,
	}
	rand.Shuffle(len(terrains), func(i, j int) {
		terrains[i], terrains[j] = terrains[j], terrains[i]
	})
	board := &Board{
		Tiles:        make(map[TileCoord]Tile),
		Roads:        make(map[PathCoord]int),
		Settlements:  make(map[CrossCoord]int),
		CityUpgrades: make(map[CrossCoord]int),
		PlayerColors: playerColors,
	}
	for _, tileCoord := range orderedTiles {
		_, valid := NewTileCoord(tileCoord.X, tileCoord.Y)
		if !valid {
			panic(fmt.Sprintf("tileCoord %v not valid", tileCoord))
		}
		terr := terrains[0]
		terrains = terrains[1:]
		if terr == TerrainDesert {
			board.Tiles[tileCoord] = Tile{Terrain: terr, DiceNumber: 0}
		} else {
			board.Tiles[tileCoord] = Tile{Terrain: terr, DiceNumber: diceNumbers[0]}
			diceNumbers = diceNumbers[1:]
		}
	}
	return board
}
