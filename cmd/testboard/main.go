package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/VichoReyes/el-poblador/board"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	ports := flag.Bool("ports", false, "preview the board with its nine coastal ports")
	pieces := flag.Bool("pieces", false, "add sample roads, settlements, a city, and a cursor to the port preview")
	emoji := flag.Bool("emoji", true, "use emojis instead of text abbreviations")
	flag.Parse()
	view := board.BoardView{UseEmoji: *emoji}
	if *ports {
		previewPorts(view, *pieces)
		return
	}

	fmt.Println()
	fmt.Println("=== Tile Rendering Test ===")

	// Test different tile types
	tiles := []board.Tile{
		{Terrain: board.TerrainWood, DiceNumber: 6},
		{Terrain: board.TerrainBrick, DiceNumber: 8},
		{Terrain: board.TerrainOre, DiceNumber: 12},
		{Terrain: board.TerrainWheat, DiceNumber: 4},
		{Terrain: board.TerrainSheep, DiceNumber: 10},
		{Terrain: board.TerrainDesert, DiceNumber: 0}, // Desert has no dice
	}

	for i, tile := range tiles {
		fmt.Printf("Tile %d (%s, dice: %d):\n", i+1, tile.Terrain.String(), tile.DiceNumber)
		// Test robber rendering on tile 3 (Ore, dice 12)
		hasRobber := (i == 2)
		if hasRobber {
			fmt.Printf("  (with robber)\n")
		}
		rendered := tile.RenderTile(false, hasRobber, view)
		for i, line := range rendered {
			if i == 0 || i == len(rendered)-1 {
				fmt.Printf(" %s\n", line)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Board Component Test ===")
	fmt.Println("Testing board printing functionality:")
	fmt.Println()

	boardInstance := board.NewChaoticBoard()

	// Place robber on a tile to test robber rendering
	robberCoord, valid := board.NewTileCoord(1, 2)
	if valid {
		boardInstance.PlaceRobber(robberCoord)
		fmt.Printf("Robber placed on the board for testing\n\n")
	}

	lines := boardInstance.Print(view)

	total := strings.Join(lines, "\n")
	fmt.Println(total)
	fmt.Printf("\nBoard size: %d, %d\n", lipgloss.Width(total), lipgloss.Height(total))
}

func previewPorts(view board.BoardView, pieces bool) {
	colors := map[int]lipgloss.AdaptiveColor{
		0: {Light: "#1565C0", Dark: "#42A5F5"},
		1: {Light: "#C62828", Dark: "#EF5350"},
		2: {Light: "#2E7D32", Dark: "#66BB6A"},
		3: {Light: "#F57C00", Dark: "#FFA726"},
	}
	b := board.NewLegalBoard(colors)
	for coord, tile := range b.Tiles {
		if tile.Terrain == board.TerrainDesert {
			b.PlaceRobber(coord)
		}
	}
	if pieces {
		// These pieces demonstrate rendering at both ends of every port; they
		// are deliberately not a legal settlement arrangement.
		for path := range b.Ports {
			owner := (path.From.X + path.From.Y) % 4
			b.Roads[path] = owner
			b.Settlements[path.From], b.Settlements[path.To] = owner, owner
		}
		city := board.CrossCoord{X: 3, Y: 0}
		b.CityUpgrades[city] = b.Settlements[city]
		cursor := board.CrossCoord{X: 0, Y: 4}
		view.CrossCursor = &cursor
	}
	text := strings.Join(b.Print(view), "\n")
	fmt.Println(text)
	fmt.Printf("\nBoard size: %d × %d\n", lipgloss.Width(text), lipgloss.Height(text))
}
