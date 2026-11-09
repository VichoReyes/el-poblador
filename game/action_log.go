package game

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogAction concatenates text parts. Player pointers contribute their names,
// with color applied only when the log is displayed to a particular viewer.
// Like other game mutations, callers must hold the game lock.
func (g *Game) LogAction(parts ...any) {
	entry := make(RichText, 0, len(parts))
	for _, part := range parts {
		if player, ok := part.(*Player); ok {
			entry = append(entry, playerText(player, player.Name))
		} else {
			entry = append(entry, TextPart{Text: fmt.Sprint(part)})
		}
	}

	g.ActionLog = append([]RichText{entry}, g.ActionLog...)
	if len(g.ActionLog) > 15 {
		g.ActionLog = g.ActionLog[:15]
	}
}

func (g *Game) renderActionLog(r *lipgloss.Renderer) string {
	lines := make([]string, len(g.ActionLog))
	for i, entry := range g.ActionLog {
		lines[i] = entry.Render(r)
	}
	return strings.Join(lines, "\n")
}
