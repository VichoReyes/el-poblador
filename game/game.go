package game

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/VichoReyes/el-poblador/board"

	"github.com/charmbracelet/lipgloss"
)

type Phase interface {
	ConfirmTurn() (Phase, EnterMode)
	PhaseView
	BoardCursor() interface{}
}

type PhaseView interface {
	MoveCursor(direction string)
	HelpText() string
	Menu() string
}

type EnterMode int

const (
	EnterPush EnterMode = iota
	EnterReplace
	EnterReset
)

// SpectatorPhase is the interface for non-turn holder phases
// All spectator phases should store their playerID internally
type SpectatorPhase interface {
	ConfirmSpectator() (SpectatorPhase, EnterMode)
	PhaseView
}

func (g *Game) LogAction(action string) {
	g.ActionLog = append([]string{action}, g.ActionLog...)
	if len(g.ActionLog) > 15 {
		g.ActionLog = g.ActionLog[:15]
	}
}

// Main game state. Has mutex
type Game struct {
	m        sync.RWMutex
	Board    *board.Board
	Players  []Player
	LastDice [2]int
	// stack backing the turn holder state
	phaseStack []Phase
	// stacks backing the other players' states
	spectatorPhaseStacks map[int][]SpectatorPhase
	// who is the turn holder
	PlayerTurn  int
	DevCardDeck []DevCard
	ActionLog   []string
	TradeOffers []TradeOffer
	shouldQuit  bool
}

// reset states, and return first Phase of next turn
func (g *Game) NextTurn() Phase {
	g.spectatorPhaseStacks = nil
	g.TradeOffers = nil
	g.PlayerTurn++
	g.PlayerTurn %= len(g.Players)
	nextPlayer := &g.Players[g.PlayerTurn]
	g.LogAction(fmt.Sprintf("Turn passed to %s", nextPlayer.RenderName()))
	return PhaseDiceRoll(g)
}

const sidebarWidth = 34

type PrintOptions struct {
	Width, Height int
	// RequestPlayer is the player that the user is playing as.
	// If nil, the game will render from the perspective of the turn holder.
	RequestPlayer *int
	// TwoColumnCycle and OneColumnCycle control which columns are visible in responsive layouts.
	TwoColumnCycle, OneColumnCycle int
	// UseEmoji controls whether to render terrain with emojis or text abbreviations.
	UseEmoji bool
}

// PrintBoard renders the board without requiring players or an active phase.
func (g *Game) PrintBoard(view board.BoardView) string {
	g.m.RLock()
	defer g.m.RUnlock()
	return strings.Join(g.Board.Print(view), "\n")
}

// Game.Print renders the whole game, with relevant options like perspective
func (g *Game) Print(opts PrintOptions) string {
	g.m.RLock()
	defer g.m.RUnlock()
	if opts.Width == 0 {
		opts.Width = 80
	}
	if opts.Height == 0 {
		opts.Height = 32
	}
	playerPerspective := g.playerPerspective(opts.RequestPlayer)
	margin := lipgloss.NewStyle().Margin(1)

	help := g.helpText(opts.Width, playerPerspective)
	sidebar := g.buildSidebar(playerPerspective, margin, opts.UseEmoji)

	// Construct BoardView from current phase cursor
	view := board.BoardView{
		UseEmoji: opts.UseEmoji,
	}
	phase := g.Phase()
	if phase == nil {
		return ""
	}
	cursor := phase.BoardCursor()
	switch c := cursor.(type) {
	case board.TileCoord:
		view.TileCursor = &c
	case board.CrossCoord:
		view.CrossCursor = &c
	}

	boardLines := g.Board.Print(view)
	boardContent := strings.Join(boardLines, "\n")
	boardWidth := lipgloss.Width(boardContent)
	boardHeight := lipgloss.Height(boardContent)
	actionLogDecor := 4 // border + margin
	minActionLogWidth := 30
	threeColMin := boardWidth + sidebarWidth + minActionLogWidth + actionLogDecor
	twoColMin := boardWidth + sidebarWidth

	var actionLogWidth int
	if opts.Width >= threeColMin {
		actionLogWidth = opts.Width - boardWidth - sidebarWidth - 2
	} else {
		actionLogWidth = boardWidth - actionLogDecor
	}

	actionLogStyle := margin.Border(lipgloss.NormalBorder()).Width(actionLogWidth).Height(boardHeight - actionLogDecor)
	actionLogContent := strings.Join(g.ActionLog, "\n")
	actionLogRendered := actionLogStyle.Render(actionLogContent)

	var layout string
	if opts.Width >= threeColMin {
		layout = lipgloss.JoinHorizontal(lipgloss.Top, actionLogRendered, boardContent, sidebar)
	} else if opts.Width >= twoColMin {
		if opts.TwoColumnCycle == 0 {
			layout = lipgloss.JoinHorizontal(lipgloss.Top, boardContent, sidebar)
		} else {
			layout = lipgloss.JoinHorizontal(lipgloss.Top, actionLogRendered, sidebar)
		}
	} else {
		switch opts.OneColumnCycle {
		case 0:
			layout = sidebar
		case 1:
			layout = boardContent
		case 2:
			layout = actionLogRendered
		}
	}

	availableHeight := opts.Height - lipgloss.Height(help)
	mainContent := lipgloss.Place(opts.Width, availableHeight, lipgloss.Center, lipgloss.Center, layout)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, help)
}

func (g *Game) buildSidebar(playerPerspective int, margin lipgloss.Style, useEmoji bool) string {
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

	myPlayer := &g.Players[playerPerspective]

	// Build resource table: resource types on top row, counts on bottom row
	var resourceRow, countRow strings.Builder
	for i, resource := range board.RESOURCE_TYPES {
		if i > 0 {
			resourceRow.WriteString("  ")
			countRow.WriteString("  ")
		}
		resourceRow.WriteString(resource.Display(useEmoji))
		countRow.WriteString(fmt.Sprintf("%2d", myPlayer.Resources[resource]))
	}

	myResources := []string{
		"Your resources:",
		resourceRow.String(),
		countRow.String(),
		"",
		fmt.Sprintf("Dev Cards: %d", myPlayer.TotalDevCards()),
	}
	myResourcesStr := margin.Render(strings.Join(myResources, "\n"))

	var phaseSidebar string
	if playerPerspective == g.PlayerTurn {
		if phase := g.Phase(); phase != nil {
			phaseSidebar = margin.Render(phase.Menu())
		}
	} else {
		if spectatorPhase, ok := g.SpectatorPhase(playerPerspective); ok {
			phaseSidebar = margin.Render(spectatorPhase.Menu())
		}
	}

	sidebar := lipgloss.JoinVertical(lipgloss.Left, dice, otherPlayers, myResourcesStr, phaseSidebar)
	return lipgloss.NewStyle().Width(sidebarWidth).Render(sidebar)
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
		if phase := g.Phase(); phase != nil {
			helpMsg = fmt.Sprintf("%s's turn. %s", turnPlayer.Render(turnPlayer.Name), phase.HelpText())
		} else {
			helpMsg = fmt.Sprintf("%s's turn.", turnPlayer.Render(turnPlayer.Name))
		}
	} else {
		// Spectator sees their spectator phase help text
		if spectatorPhase, ok := g.SpectatorPhase(playerPerspective); ok {
			helpMsg = fmt.Sprintf("%s's turn. %s", turnPlayer.Render(turnPlayer.Name), spectatorPhase.HelpText())
		} else {
			// No spectator phase yet (before dice roll)
			helpMsg = fmt.Sprintf("%s's turn.", turnPlayer.Render(turnPlayer.Name))
		}
	}

	renderedHelp := lipgloss.PlaceHorizontal(width, lipgloss.Center, helpMsg)
	return renderedHelp
}

func (g *Game) MakeBoard() {
	// TODO: make it so colors order isn't always the same,
	// maybe even allow picking player color.
	// Distinctive colors that work well on both light and dark backgrounds
	colors := []lipgloss.AdaptiveColor{
		{Light: "#1565C0", Dark: "#42A5F5"}, // Blue
		{Light: "#C62828", Dark: "#EF5350"}, // Red
		{Light: "#F57C00", Dark: "#FFB74D"}, // Orange
		{Light: "#6A1B9A", Dark: "#AB47BC"}, // Purple
	}
	// Create player color map for board rendering
	playerColors := make(map[int]lipgloss.AdaptiveColor)
	for i, c := range colors {
		playerColors[i] = c
	}
	g.Board = board.NewLegalBoard(playerColors)
}

func (g *Game) Start(playerNames []string) {
	g.StartOptions(playerNames, true, true)
}

func (g *Game) StartOptions(playerNames []string, shufflePlayers, initBoard bool) {
	if len(playerNames) < 3 || len(playerNames) > 4 {
		panic("Game must have 3-4 players")
	}
	if shufflePlayers {
		rand.Shuffle(len(playerNames), func(i, j int) {
			playerNames[i], playerNames[j] = playerNames[j], playerNames[i]
		})
	}
	if initBoard {
		g.MakeBoard()
	} else if g.Board == nil {
		panic("Board should've been initialized")
	}
	g.Players = make([]Player, len(playerNames))
	for i, name := range playerNames {
		g.Players[i] = Player{
			Name:                 name,
			Color:                g.Board.PlayerColors[i],
			Resources:            make(map[board.ResourceType]int),
			HiddenDevCards:       make([]DevCard, 0),
			PlayedDevCards:       make([]DevCard, 0),
			CitiesAvailable:      4,
			SettlementsAvailable: 5,
			RoadsAvailable:       15,
		}
	}
	g.PlayerTurn = 0
	g.phaseStack = []Phase{PhaseInitialSettlements(g, true)}
	g.DevCardDeck = shuffleDevCards()
	g.ActionLog = make([]string, 0, 15)
}

func (g *Game) MoveCursor(direction string, requestPlayer *int) {
	g.m.Lock()
	defer g.m.Unlock()
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		// Turn holder moves cursor in their phase
		if phase := g.Phase(); phase != nil {
			phase.MoveCursor(direction)
		}
	} else {
		// Spectator moves cursor in their spectator phase
		if spectatorPhase, ok := g.SpectatorPhase(playerPerspective); ok {
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
	if p, ok := g.Phase().(*phaseInitialSettlements); ok {
		p.cursorCross = find()
	}
}

func (g *Game) ConfirmAction(requestPlayer *int) {
	g.m.Lock()
	defer g.m.Unlock()
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		if phase := g.Phase(); phase != nil {
			next, mode := phase.ConfirmTurn()
			g.applyTurnTransition(next, mode)
		}
	} else {
		if spectatorPhase, ok := g.SpectatorPhase(playerPerspective); ok {
			next, mode := spectatorPhase.ConfirmSpectator()
			g.applySpectatorTransition(playerPerspective, next, mode)
		}
	}
}

func (g *Game) CancelAction(requestPlayer *int) {
	g.m.Lock()
	defer g.m.Unlock()
	playerPerspective := g.playerPerspective(requestPlayer)

	if playerPerspective == g.PlayerTurn {
		g.ensureTurnStack()
		if len(g.phaseStack) > 1 {
			g.phaseStack = g.phaseStack[:len(g.phaseStack)-1]
		}
	} else {
		if _, ok := g.SpectatorPhase(playerPerspective); ok {
			g.ensureSpectatorStack(playerPerspective)
			stack := g.spectatorPhaseStacks[playerPerspective]
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
				g.spectatorPhaseStacks[playerPerspective] = stack
			}
		}
	}
}

func (g *Game) ensureTurnStack() {
	if len(g.phaseStack) == 0 {
		return
	}
}

func (g *Game) applyTurnTransition(next Phase, mode EnterMode) {
	if next == nil {
		return
	}
	g.ensureTurnStack()
	switch mode {
	case EnterPush:
		g.phaseStack = append(g.phaseStack, next)
	case EnterReplace:
		if len(g.phaseStack) == 0 {
			g.phaseStack = []Phase{next}
		} else {
			g.phaseStack[len(g.phaseStack)-1] = next
		}
	case EnterReset:
		g.phaseStack = []Phase{next}
	default:
		panic("unknown turn enter mode")
	}
}

func (g *Game) ensureSpectatorStack(playerID int) {
	if g.spectatorPhaseStacks == nil {
		g.spectatorPhaseStacks = make(map[int][]SpectatorPhase)
	}
	stack := g.spectatorPhaseStacks[playerID]
	if len(stack) == 0 {
		return
	}
}

func (g *Game) applySpectatorTransition(playerID int, next SpectatorPhase, mode EnterMode) {
	if next == nil {
		return
	}
	g.ensureSpectatorStack(playerID)
	stack := g.spectatorPhaseStacks[playerID]
	switch mode {
	case EnterPush:
		stack = append(stack, next)
	case EnterReplace:
		if len(stack) == 0 {
			stack = []SpectatorPhase{next}
		} else {
			stack[len(stack)-1] = next
		}
	case EnterReset:
		stack = []SpectatorPhase{next}
	default:
		panic("unknown spectator enter mode")
	}
	g.spectatorPhaseStacks[playerID] = stack
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

// getPlayerID returns the player ID for a given player.
// panics if it can't find the player.
func (g *Game) getPlayerID(player *Player) int {
	for i := range g.Players {
		if &g.Players[i] == player {
			return i
		}
	}
	panic("Player not found")
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
	g.m.RLock()
	defer g.m.RUnlock()
	return g.shouldQuit
}

func (g *Game) Phase() Phase {
	if len(g.phaseStack) == 0 {
		return nil
	}
	return g.phaseStack[len(g.phaseStack)-1]
}

func (g *Game) SetPhase(phase Phase) {
	if phase == nil {
		g.phaseStack = nil
		return
	}
	g.phaseStack = []Phase{phase}
}

func (g *Game) SpectatorPhase(playerID int) (SpectatorPhase, bool) {
	stack := g.spectatorPhaseStacks[playerID]
	if len(stack) == 0 {
		return nil, false
	}
	return stack[len(stack)-1], true
}

func (g *Game) SetSpectatorPhase(playerID int, phase SpectatorPhase) {
	if phase == nil {
		if g.spectatorPhaseStacks != nil {
			delete(g.spectatorPhaseStacks, playerID)
		}
		return
	}
	if g.spectatorPhaseStacks == nil {
		g.spectatorPhaseStacks = make(map[int][]SpectatorPhase)
	}
	g.spectatorPhaseStacks[playerID] = []SpectatorPhase{phase}
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
