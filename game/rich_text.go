package game

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TextPart keeps text together with its style, applied per viewer.
type TextPart struct {
	Text  string
	style lipgloss.Style // presentation state is transient
}

type RichText []TextPart

func (text RichText) String() string {
	var result strings.Builder
	for _, part := range text {
		result.WriteString(part.Text)
	}
	return result.String()
}

// Render applies each part's style using the viewer's renderer.
func (text RichText) Render(r *lipgloss.Renderer) string {
	var result strings.Builder
	for _, part := range text {
		result.WriteString(part.style.Renderer(r).Render(part.Text))
	}
	return result.String()
}

func plainText(s string) RichText { return RichText{{Text: s}} }

func playerText(player *Player, s string) TextPart {
	return TextPart{Text: s, style: lipgloss.NewStyle().Foreground(player.Color)}
}
