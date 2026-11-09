package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Bubbles creates its default styles with the process renderer. Rebind those
// styles before rendering an SSH session, preserving the component's defaults.
func bindStyles(r *lipgloss.Renderer, styles ...*lipgloss.Style) {
	for _, style := range styles {
		*style = style.Renderer(r)
	}
}

func bindInputStyles(m *textinput.Model, r *lipgloss.Renderer) {
	bindStyles(r, &m.PromptStyle, &m.TextStyle, &m.PlaceholderStyle,
		&m.CompletionStyle, &m.CursorStyle, &m.Cursor.Style, &m.Cursor.TextStyle)
}

func bindListStyles(m *list.Model, r *lipgloss.Renderer) {
	s := &m.Styles
	bindStyles(r, &s.TitleBar, &s.Title, &s.Spinner, &s.FilterPrompt, &s.FilterCursor,
		&s.DefaultFilterCharacterMatch, &s.StatusBar, &s.StatusEmpty,
		&s.StatusBarActiveFilter, &s.StatusBarFilterCount, &s.NoItems,
		&s.PaginationStyle, &s.HelpStyle, &s.ActivePaginationDot,
		&s.InactivePaginationDot, &s.ArabicPagination, &s.DividerDot)
	h := &m.Help.Styles
	bindStyles(r, &h.Ellipsis, &h.ShortKey, &h.ShortDesc, &h.ShortSeparator,
		&h.FullKey, &h.FullDesc, &h.FullSeparator)
	bindInputStyles(&m.FilterInput, r)
	// Pagination dots are already rendered strings in the list constructor.
	m.Paginator.ActiveDot = s.ActivePaginationDot.String()
	m.Paginator.InactiveDot = s.InactivePaginationDot.String()
}
