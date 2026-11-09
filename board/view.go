package board

import "github.com/charmbracelet/lipgloss"

// BoardView contains both rendering preferences and UI interaction state
// for displaying the board. This is transient view state that should not
// be serialized with game state.
type BoardView struct {
	Renderer    *lipgloss.Renderer // nil uses the local terminal renderer
	UseEmoji    bool
	TileCursor  *TileCoord  // nil when no tile is selected
	CrossCursor *CrossCoord // nil when no crossing is selected
}

func (v BoardView) newStyle() lipgloss.Style {
	if v.Renderer != nil {
		return v.Renderer.NewStyle()
	}
	return lipgloss.NewStyle()
}
