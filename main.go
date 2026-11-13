package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"os"

	"github.com/VichoReyes/el-poblador/board"
	"github.com/VichoReyes/el-poblador/game"
	"github.com/VichoReyes/el-poblador/internal/tui"
	"github.com/charmbracelet/log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

func loadGameState(filename string) (*game.Game, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	var g game.Game
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&g); err != nil {
		return nil, fmt.Errorf("decoding failed: %w", err)
	}

	// Restore phase to PhaseDiceRoll
	g.SetPhase(game.PhaseDiceRoll(&g))

	return &g, nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  el_poblador new [options] <player1> <player2> <player3> [player4]")
	fmt.Println("  el_poblador load <filename.gob>")
	fmt.Println("  el_poblador serve")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  new    Start a new game with 3-4 players")
	fmt.Println("  load   Load a saved game from file")
	fmt.Println("  serve  Start an SSH server for multiplayer")
	fmt.Println()
	fmt.Println("Run 'el_poblador <command> -help' for command-specific options.")
}

func main() {
	// Handle Termux bug where os.Args[1] contains the executable path
	args := os.Args[1:]
	if len(args) > 0 && len(args[0]) > 0 && (args[0] == os.Args[0] || args[0][0] == '/') {
		// Skip the executable path if it appears as first arg
		args = args[1:]
	}

	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "new":
		// logFile, err := os.OpenFile("poblador.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		// if err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }
		// defer logFile.Close()
		// log.SetOutput(logFile)
		newCmd := flag.NewFlagSet("new", flag.ExitOnError)
		rich := newCmd.Bool("rich", false, "Start all players with 20 of each resource (for testing)")
		boardDebug := newCmd.Bool("board-debug", false, "Debug mode for board coordinates")
		noEmoji := newCmd.Bool("no-emoji", false, "Don't use emojis, they break board alignment on some terminals")
		newCmd.Usage = func() {
			fmt.Println("Usage: el_poblador new [options] <player1> <player2> <player3> [player4]")
			fmt.Println()
			fmt.Println("Options:")
			newCmd.PrintDefaults()
		}
		newCmd.Parse(args[1:])

		playerNames := newCmd.Args()
		if len(playerNames) < 3 || len(playerNames) > 4 {
			fmt.Println("Error: 'new' command requires 3-4 player names")
			fmt.Println()
			newCmd.Usage()
			os.Exit(1)
		}

		g := &game.Game{}
		g.Start(playerNames)

		if *boardDebug {
			g.SetPhase(game.PhaseDebugTiles(g))
		}

		if *rich {
			for i := range g.Players {
				for _, res := range board.RESOURCE_TYPES {
					g.Players[i].Resources[res] = 20
				}
			}
		}
		p := tea.NewProgram(tui.NewLocalConn(g, !*noEmoji), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

	case "load":
		loadCmd := flag.NewFlagSet("load", flag.ExitOnError)
		noEmoji := loadCmd.Bool("no-emoji", false, "Don't use emojis, they break board alignment on some terminals")
		loadCmd.Usage = func() {
			fmt.Println("Usage: el_poblador load [options] <filename.gob>")
			fmt.Println()
			fmt.Println("Options:")
			loadCmd.PrintDefaults()
		}
		loadCmd.Parse(args[1:])

		if loadCmd.NArg() != 1 {
			fmt.Println("Error: 'load' command requires a filename")
			fmt.Println()
			loadCmd.Usage()
			os.Exit(1)
		}

		loadedGame, err := loadGameState(loadCmd.Arg(0))
		if err != nil {
			fmt.Printf("Failed to load game: %v\n", err)
			os.Exit(1)
		}
		p := tea.NewProgram(tui.NewLocalConn(loadedGame, !*noEmoji), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := serveCmd.String("addr", "127.0.0.1:2222", "address, including port")
		keyPath := serveCmd.String("keyPath", ".ssh/id_ed25519", "path to private SSH key")
		dummy := serveCmd.Bool(
			"dummy", false,
			"make a local-like game per connection instead of multiplayer")
		serveCmd.Usage = func() {
			fmt.Println("Options:")
			serveCmd.PrintDefaults()
		}
		serveCmd.Parse(args[1:])
		log.Info("starting server", "address", *addr)
		var gameMiddleware wish.Middleware
		if *dummy {
			gameMiddleware = bubbletea.Middleware(tui.DummyHandler)
		} else {
			gameMiddleware = tui.MultiplayerMiddleware()
		}
		server, err := wish.NewServer(
			wish.WithAddress(*addr),
			wish.WithHostKeyPath(*keyPath),
			wish.WithMiddleware(
				gameMiddleware,
				activeterm.Middleware(),
				logging.StructuredMiddleware(),
			),
		)
		if err != nil {
			log.Fatal("server failed", "error", err)
		}
		log.Fatal("server stopped", "error", server.ListenAndServe())

	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		fmt.Println()
		printUsage()
		os.Exit(1)
	}
}
