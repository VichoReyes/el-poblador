package board

// BoardView contains both rendering preferences and UI interaction state
// for displaying the board. This is transient view state that should not
// be serialized with game state.
type BoardView struct {
	TileCursor  *TileCoord  // nil when no tile is selected
	CrossCursor *CrossCoord // nil when no crossing is selected
}
