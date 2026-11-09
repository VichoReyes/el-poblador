package main

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/termenv"
)

func terminalRenderer(profile termenv.Profile, dark bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	r.SetHasDarkBackground(dark)
	return r
}

func TestSessionColorsFollowViewer(t *testing.T) {
	// The host has no terminal or colors, as with a background service.
	previous := lipgloss.DefaultRenderer()
	lipgloss.SetDefaultRenderer(terminalRenderer(termenv.Ascii, true))
	t.Cleanup(func() { lipgloss.SetDefaultRenderer(previous) })

	registry := newLobbyRegistry()
	dark := newSessionModel(registry, "alice", 140, 40, terminalRenderer(termenv.TrueColor, true))
	light := newSessionModel(registry, "bob", 140, 40, terminalRenderer(termenv.TrueColor, false))
	plain := newSessionModel(registry, "carol", 140, 40, terminalRenderer(termenv.Ascii, true))
	sessions := []*SessionModel{dark, light, plain}
	for _, session := range sessions {
		t.Cleanup(session.Close)
		if got := strings.Contains(session.View(), "\x1b["); got != (session != plain) {
			t.Fatalf("%s: lobby browser colors do not follow viewer", session.username)
		}
	}
	createTestLobby(t, dark, "Colors")
	for _, session := range sessions[1:] {
		session.Update(TickMsg{})
		navigate(t, session, tea.KeyEnter)
	}
	checkViews := func() {
		t.Helper()
		for _, session := range sessions {
			view := session.View()
			if session == plain {
				if strings.Contains(view, "\x1b[") {
					t.Fatal("colorless client received ANSI styles")
				}
				continue
			}
			// Forest terrain has distinct light and dark RGB colors.
			color, otherColor := "#4CAF50", "#2D5016"
			if session == light {
				color, otherColor = otherColor, color
			}
			want := session.renderer.NewStyle().Foreground(lipgloss.Color(color)).Render("/‾‾‾‾\\")
			other := session.renderer.NewStyle().Foreground(lipgloss.Color(otherColor)).Render("/‾‾‾‾\\")
			if !strings.Contains(view, want) || strings.Contains(view, other) {
				t.Fatalf("%s: board does not use viewer's background palette", session.username)
			}
		}
	}
	checkViews() // waiting room preview
	for _, session := range sessions {
		session.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	for _, session := range sessions {
		session.Update(TickMsg{})
		if conn, ok := session.active.(*GameConn); !ok || conn.renderer != session.renderer {
			t.Fatalf("%s: game transition lost viewer renderer", session.username)
		}
	}
	checkViews() // active game

	// Concurrent rendering of one game must never switch another user's palette.
	var wg sync.WaitGroup
	for _, session := range sessions {
		wg.Add(1)
		go func(session *SessionModel) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				session.View()
			}
		}(session)
	}
	wg.Wait()
	checkViews()
	if lipgloss.ColorProfile() != termenv.Ascii {
		t.Fatal("session rendering changed the process color profile")
	}
}

// A terminal-protocol fixture exercises the real Wish renderer factory through
// the SSH game handler, including PTY TERM and the client's OSC 11 response.
type colorSession struct {
	ssh.Session
	input  *strings.Reader
	output bytes.Buffer
	term   string
	env    []string
}

type colorContext struct{ ssh.Context }

func (colorContext) Value(any) any                  { return nil }
func (s *colorSession) Context() ssh.Context        { return colorContext{} }
func (s *colorSession) Environ() []string           { return s.env }
func (s *colorSession) Read(p []byte) (int, error)  { return s.input.Read(p) }
func (s *colorSession) Write(p []byte) (int, error) { return s.output.Write(p) }
func (s *colorSession) Stderr() io.ReadWriter       { return &s.output }
func (s *colorSession) Pty() (ssh.Pty, <-chan ssh.Window, bool) {
	return ssh.Pty{Term: s.term, Window: ssh.Window{Width: 140, Height: 40}}, nil, true
}

func TestSSHRendererUsesClientTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("COLORTERM", "")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLORFGBG", "0;15")
	for _, tc := range []struct {
		name, term, background string
		env                    []string
		profile                termenv.Profile
		dark                   bool
	}{
		{"dark truecolor", "xterm-kitty", "0000/0000/0000", nil, termenv.TrueColor, true},
		{"light truecolor", "xterm-kitty", "ffff/ffff/ffff", nil, termenv.TrueColor, false},
		{"256 colors", "xterm-256color", "0000/0000/0000", nil, termenv.ANSI256, true},
		{"16 colors", "xterm", "0000/0000/0000", nil, termenv.ANSI, true},
		{"client no color", "xterm-kitty", "0000/0000/0000", []string{"NO_COLOR=1"}, termenv.Ascii, true},
		{"dumb", "dumb", "", nil, termenv.Ascii, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &colorSession{term: tc.term, env: tc.env,
				input: strings.NewReader("\x1b]11;rgb:" + tc.background + "\x1b\\\x1b[?1;2c")}
			model, _ := dummyHandler(s)
			r := model.(*GameConn).renderer
			if r.ColorProfile() != tc.profile {
				t.Fatalf("profile = %v, want %v", r.ColorProfile(), tc.profile)
			}
			if tc.term != "dumb" && r.HasDarkBackground() != tc.dark {
				t.Fatal("background detection ignored the client's OSC response")
			}
			view := model.View()
			if got := strings.Contains(view, "\x1b["); got != (tc.profile != termenv.Ascii) {
				t.Fatal("SSH game colors do not match the client profile")
			}
			if tc.term != "dumb" && !strings.Contains(s.output.String(), "\x1b]11;") {
				t.Fatal("client was not queried for its background color")
			}
		})
	}
}

func TestLobbyFormRetainsSessionRenderer(t *testing.T) {
	for _, profile := range []termenv.Profile{termenv.Ascii, termenv.TrueColor} {
		session := newSessionModel(newLobbyRegistry(), "alice", 80, 32, terminalRenderer(profile, false))
		navigate(t, session, tea.KeyRunes, 'n')
		if got := strings.Contains(session.View(), "\x1b["); got != (profile != termenv.Ascii) {
			t.Fatal("lobby form did not use the session renderer")
		}
		navigate(t, session, tea.KeyEsc)
		if got := strings.Contains(session.View(), "\x1b["); got != (profile != termenv.Ascii) {
			t.Fatal("returning to the browser lost the session renderer")
		}
	}
}
