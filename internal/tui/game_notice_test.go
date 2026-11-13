package tui

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/VichoReyes/el-poblador/game"
	tea "github.com/charmbracelet/bubbletea"
)

type noticeOutput struct {
	bytes.Buffer
	printed chan struct{}
	once    sync.Once
}

func (w *noticeOutput) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if bytes.Contains(p, []byte("TEST-SAVE-CODE")) {
		w.once.Do(func() { close(w.printed) })
	}
	return n, err
}

type noticeModel struct{ *GameConn }

func (m noticeModel) Init() tea.Cmd { return func() tea.Msg { return TickMsg{} } }

func TestSaveNoticePersistsOutsideAlternateScreen(t *testing.T) {
	var g game.Game
	g.MakeBoard()
	g.StartOptions([]string{"alice", "bob", "carol"}, false, false)
	g.SetSaveHandler(func(*game.Game) (string, error) { return "Resume TEST-SAVE-CODE\nCtrl+C to disconnect.", nil })
	g.SetPhase(game.PhaseDiceRoll(&g))
	g.MoveCursor("down", nil)
	g.MoveCursor("down", nil)
	g.ConfirmAction(nil)
	output := &noticeOutput{printed: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p := tea.NewProgram(noticeModel{&GameConn{game: &g}}, tea.WithAltScreen(), tea.WithInput(nil), tea.WithOutput(output), tea.WithContext(ctx), tea.WithoutSignalHandler())
	go func() {
		select {
		case <-output.printed:
			p.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		case <-ctx.Done():
		}
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	exit := strings.Index(text, "\x1b[?1049l")
	notice := strings.Index(text, "TEST-SAVE-CODE")
	if exit < 0 || notice < exit {
		t.Fatalf("notice was not printed after leaving alternate screen: %q", text)
	}
	if strings.Count(text, "TEST-SAVE-CODE") != 1 {
		t.Fatalf("notice repeated: %q", text)
	}
}
