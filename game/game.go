package game

import (
	"fmt"
	"git.sr.ht/~vicho/el_poblador/board"
	"math/rand/v2"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Phase interface {
	Confirm() Phase
	MoveCursor(direction string)
	BoardCursor() interface{}
	HelpText() string
}

type PhaseWithMenu interface {
	Phase
	Menu() string
}

type PhaseCancelable interface {
	Phase
	Cancel() Phase
}

// SpectatorPhase is the interface for non-turn holder phases
// All spectator phases should store their playerID internally
type SpectatorPhase interface {
	Confirm(game *Game) SpectatorPhase
	Cancel(game *Game) SpectatorPhase
	MoveCursor(direction string)
	HelpText() string
	Menu() string
}

func (g *Game) LogAction(action string) {
	g.ActionLog = append([]string{action}, g.ActionLog...)
	if len(g.ActionLog) > 15 {
		g.ActionLog = g.ActionLog[:15]
	}
}

// Main game state
type Game struct {
	Board    *board.Board
	Players  []Player
	LastDice [2]int
	// the turn holder's state
	phase Phase
	// the other players' states
	spectatorPhases map[int]SpectatorPhase
	// who is the turn holder
	PlayerTurn  int
	DevCardDeck []DevCard
	ActionLog   []string
	TradeOffers []TradeOffer
	shouldQuit  bool
}

// reset states, and return first Phase of next turn
func (g *Game) NextTurn() Phase {
	g.spectatorPhases = nil
	g.TradeOffers = nil
	g.PlayerTurn++
	g.PlayerTurn %= len(g.Players)
	nextPlayer := &g.Players[g.PlayerTurn]
	g.LogAction(fmt.Sprintf("Turn passed to %s", nextPlayer.RenderName()))
	return PhaseDiceRoll(g)
}

// requestPlayer is the player that the user is playing as.
// If nil, the game will render from the perspective of the turn holder.
// twoColumnCycle and oneColumnCycle control which columns are visible in responsive layouts.
func (g *Game) Print(width, height int, requestPlayer *int, twoColumnCycle, oneColumnCycle int) string {
	playerPerspective := g.playerPerspective(requestPlayer)
	margin := lipgloss.NewStyle().Margin(1)

	help := g.helpText(width, playerPerspective)
	sidebar := g.buildSidebar(playerPerspective, margin)

	boardLines := g.Board.Print(g.phase.BoardCursor())
	boardContent := strings.Join(boardLines, "\n")
	boardWidth := lipgloss.Width(boardContent)
	boardHeight := lipgloss.Height(boardContent)

	var actionLogWidth int
	if width >= 120 {
		actionLogWidth = width - boardWidth - 36
	} else {
		actionLogWidth = boardWidth - 4
	}

	actionLogStyle := margin.Border(lipgloss.NormalBorder()).Width(actionLogWidth).Height(boardHeight - 4)
	actionLogContent := strings.Join(g.ActionLog, "\n")
	actionLogRendered := actionLogStyle.Render(actionLogContent)

	var layout string
	if width >= 120 {
		layout = lipgloss.JoinHorizontal(lipgloss.Top, actionLogRendered, boardContent, sidebar)
	} else if width >= 90 {
		if twoColumnCycle == 0 {
			layout = lipgloss.JoinHorizontal(lipgloss.Top, boardContent, sidebar)
		} else {
			layout = lipgloss.JoinHorizontal(lipgloss.Top, actionLogRendered, sidebar)
		}
	} else {
		switch oneColumnCycle {
		case 0:
			layout = sidebar
		case 1:
			layout = boardContent
		case 2:
			layout = actionLogRendered
		}
	}

	availableHeight := height - lipgloss.Height(help)
	mainContent := lipgloss.Place(width, availableHeight, lipgloss.Center, lipgloss.Center, layout)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, help)
}

func (g *Game) buildSidebar(playerPerspective int, margin lipgloss.Style) string {
	var dice string
	if g.LastDice[0] != 0 {
		dice = fmt.Sprintf("Dice: %d (%d + %d)", g.LastDice[0]+g.LastDice[1], g.LastDice[0], g.LastDice[1])
	} else {
		dice = "Dice: not rolled yet"
	}
	dice = margin.Render(dice)

	var playerList []string
	for i, player := range g.Players {
		var name string
		if i == playerPerspective {
			name = fmt.Sprintf("%s (you)", player.Render(player.Name))
		} else {
			name = player.Render(player.Name)
		}
		info := player.Render(fmt.Sprintf(" has %d resources, %d dev cards", player.TotalResources(), player.TotalDevCards()))
		playerList = append(playerList, name, info)
	}
	otherPlayers := margin.Render(strings.Join(playerList, "\n"))

	myPlayer := g.Players[playerPerspective]
	myResources := []string{"Your resources:"}
	for _, resource := range board.RESOURCE_TYPES {
		myResources = append(myResources, fmt.Sprintf("%s: %d", resource, myPlayer.Resources[resource]))
	}
	myResources = append(myResources, "")
	myResources = append(myResources, fmt.Sprintf("Dev Cards: %d", myPlayer.TotalDevCards()))
	myResources = append(myResources, fmt.Sprintf("Victory Points: %d", myPlayer.VictoryPoints(g)))
	myResourcesStr := margin.Render(strings.Join(myResources, "\n"))

	var phaseSidebar string
	if playerPerspective == g.PlayerTurn {
		// Show turn holder's phase menu
		if p, ok := g.phase.(PhaseWithMenu); ok {
			phaseSidebar = margin.Render(p.Menu())
		}
	} else {
		// Show spectator's phase menu
		if spectatorPhase, ok := g.spectatorPhases[playerPerspective]; ok {
			phaseSidebar = margin.Render(spectatorPhase.Menu())
		}
	}

	sidebar := lipgloss.JoinVertical(lipgloss.Left, dice, otherPlayers, myResourcesStr, phaseSidebar)
	return lipgloss.NewStyle().Width(30).Render(sidebar)
}

func (g *Game) playerPerspective(requestPlayer *int) int {
	if requestPlayer != nil && *requestPlayer < len(g.Players) {
		return *requestPlayer
	} else {
		return g.PlayerTurn
	}
}

func (g *Game) helpText(width int, playerPerspective int) string {
	turnPlayer := &g.Players[g.PlayerTurn]
	var helpMsg string

	if playerPerspective == g.PlayerTurn {
		// Turn holder sees their phase help text
		helpMsg = fmt.Sprintf("%s's turn. %s", turnPlayer.Render(turnPlayer.Name), g.phase.HelpText())
	} else {
		// Spectator sees their spectator phase help text
		if spectatorPhase, ok := g.spectatorPhases[playerPerspective]; ok {
			helpMsg = fmt.Sprintf("%s's turn. %s", turnPlayer.Render(turnPlayer.Name), spectatorPhase.HelpText())
		} else {
			// No spectator phase yet (before dice roll)
			helpMsg = fmt.Sprintf("%s's turn.", turnPlayer.Render(turnPlayer.Name))
		}
	}

	renderedHelp := lipgloss.PlaceHorizontal(width, lipgloss.Center, helpMsg)
	return renderedHelp
}

func (g *Game) Start(playerNames []string) {
	if len(playerNames) < 3 || len(playerNames) > 4 {
		panic("Game must have 3-4 players")
	}
	// Distinctive colors that work well on both light and dark backgrounds
	colors := []lipgloss.AdaptiveColor{
		{Light: "#1565C0", Dark: "#42A5F5"}, // Blue
		{Light: "#C62828", Dark: "#EF5350"}, // Red
		{Light: "#F57C00", Dark: "#FFB74D"}, // Orange
		{Light: "#6A1B9A", Dark: "#AB47BC"}, // Purple
	}
	g.Players = make([]Player, len(playerNames))
	for i, name := range playerNames {
		g.Players[i] = Player{
			Name:                 name,
			Color:                colors[i],
			Resources:            make(map[board.ResourceType]int),
			HiddenDevCards:       make([]DevCard, 0),
			PlayedDevCards:       make([]DevCard, 0),
			CitiesAvailable:      4,
			SettlementsAvailable: 5,
			RoadsAvailable:       15,
		}
	}
	rand.Shuffle(len(g.Players), func(i, j int) {
		g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
	})
	// Create player color map for board rendering
	playerColors := make(map[int]lipgloss.AdaptiveColor)
	for i, player := range g.Players {
		playerColors[i] = player.Color
	}
	g.Board = board.NewLegalBoard(playerColors)
	g.PlayerTurn = 0
	g.phase = PhaseInitialSettlements(g, true)
	g.DevCardDeck = shuffleDevCards()
	g.ActionLog = make([]string, 0, 15)
}

func (g *Game) MoveCursor(direction string, requestPlayer *int) {
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		// Turn holder moves cursor in their phase
		g.phase.MoveCursor(direction)
	} else {
		// Spectator moves cursor in their spectator phase
		if spectatorPhase, ok := g.spectatorPhases[playerPerspective]; ok {
			spectatorPhase.MoveCursor(direction)
		}
	}
}

// Function for testing purposes: move the cursor to any valid settlement location
func (g *Game) MoveCursorToPlaceSettlement() {
	find := func() board.CrossCoord {
		for x := 0; x <= 5; x++ {
			for y := 0; y <= 10; y++ {
				coord, valid := board.NewCrossCoord(x, y)
				if valid && g.Board.CanPlaceSettlement(coord) {
					return coord
				}
			}
		}
		panic("no valid settlement location found")
	}
	if p, ok := g.phase.(*phaseInitialSettlements); ok {
		p.cursorCross = find()
	}
}

func (g *Game) ConfirmAction(requestPlayer *int) {
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		// Turn holder confirms their action
		g.phase = g.phase.Confirm()
	} else {
		// Spectator confirms their action
		if spectatorPhase, ok := g.spectatorPhases[playerPerspective]; ok {
			g.spectatorPhases[playerPerspective] = spectatorPhase.Confirm(g)
		}
	}
}

func (g *Game) CancelAction(requestPlayer *int) {
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		// Turn holder cancels their action
		if p, ok := g.phase.(PhaseCancelable); ok {
			g.phase = p.Cancel()
		}
	} else {
		// Spectator cancels their action
		if spectatorPhase, ok := g.spectatorPhases[playerPerspective]; ok {
			g.spectatorPhases[playerPerspective] = spectatorPhase.Cancel(g)
		}
	}
}

// DrawDevelopmentCard draws a card from the development card deck
func (g *Game) DrawDevelopmentCard() *DevCard {
	if len(g.DevCardDeck) == 0 {
		return nil
	}

	card := g.DevCardDeck[len(g.DevCardDeck)-1]
	g.DevCardDeck = g.DevCardDeck[:len(g.DevCardDeck)-1]
	return &card
}

// getPlayerID returns the player ID for a given player, or -1 if not found
func (g *Game) getPlayerID(player *Player) int {
	for i := range g.Players {
		if &g.Players[i] == player {
			return i
		}
	}
	return -1
}

// CheckGameEnd checks if any player has won and returns the winner, or nil if game continues
func (g *Game) CheckGameEnd() *Player {
	for i := range g.Players {
		player := &g.Players[i]
		if player.VictoryPoints(g) >= 10 {
			return player
		}
	}
	return nil
}

func (g *Game) ShouldQuit() bool {
	return g.shouldQuit
}

func (g *Game) SetPhase(phase Phase) {
	g.phase = phase
}

// AddTradeOffer creates and adds a new trade offer to the game
func (g *Game) AddTradeOffer(offererID, targetID int, offering, requesting map[board.ResourceType]int) bool {
	offer := TradeOffer{
		OffererID:  offererID,
		TargetID:   targetID,
		Offering:   offering,
		Requesting: requesting,
		Status:     OfferIsPending,
	}

	// Verify the offerer has the resources they're offering
	if !offer.canOffer(g) {
		return false
	}

	g.TradeOffers = append(g.TradeOffers, offer)
	return true
}
