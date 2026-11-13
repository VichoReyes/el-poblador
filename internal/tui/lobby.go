package tui

import (
	"errors"
	"log"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/VichoReyes/el-poblador/game"
)

const lobbyNameLimit = 40

type lobbyPlayer struct {
	isReady     bool
	playerIndex int
}

type lobbyCommon struct {
	m         sync.RWMutex
	id        uint64
	name      string
	players   map[string]*lobbyPlayer
	isStarted bool
	game      game.Game
	code      string
	resuming  bool
	seats     map[string]int
}

type lobbySnapshot struct {
	id      uint64
	name    string
	players int
	started bool
}

// Nested locking always follows registry -> one lobby. Lobby-only paths must
// not call registry methods while holding the lobby lock. Game actions and
// rendering run after releasing both locks; game initialization is published
// under the lobby lock before isStarted becomes true.
type lobbyRegistry struct {
	saves   *saveStore
	m       sync.RWMutex
	nextID  uint64
	lobbies map[uint64]*lobbyCommon
}

func newLobbyRegistry() *lobbyRegistry {
	return &lobbyRegistry{saves: &saveStore{dir: "saves"}, lobbies: make(map[uint64]*lobbyCommon)}
}

// Create reserves the name and joins the creator before publishing the lobby.
func (r *lobbyRegistry) Create(name, username string) (*lobbyCommon, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("enter a lobby name")
	}
	if utf8.RuneCountInString(name) > lobbyNameLimit {
		return nil, errors.New("lobby names must be 40 characters or fewer")
	}
	for _, ch := range name {
		if unicode.IsControl(ch) {
			return nil, errors.New("lobby names cannot contain control characters")
		}
	}
	r.m.Lock()
	defer r.m.Unlock()
	for _, lobby := range r.lobbies {
		if strings.EqualFold(lobby.name, name) {
			return nil, errors.New("a lobby with that name already exists")
		}
	}
	r.nextID++
	lobby := &lobbyCommon{
		id: r.nextID, name: name,
		players: map[string]*lobbyPlayer{username: {}},
	}
	lobby.game.MakeBoard()
	r.configureSave(lobby)
	r.lobbies[lobby.id] = lobby
	return lobby, nil
}

func (r *lobbyRegistry) List() []lobbySnapshot {
	// TODO: Copy lobby pointers under the registry lock, then release it before
	// reading individual lobbies and sorting. Each browser polls every 200 ms;
	// a busy lobby currently delays registry writes during those scans.
	r.m.RLock()
	defer r.m.RUnlock()
	lobbies := make([]lobbySnapshot, 0, len(r.lobbies))
	for _, lobby := range r.lobbies {
		lobby.m.RLock()
		if !lobby.resuming {
			lobbies = append(lobbies, lobbySnapshot{lobby.id, lobby.name, len(lobby.players), lobby.isStarted})
		}
		lobby.m.RUnlock()
	}
	sort.Slice(lobbies, func(i, j int) bool {
		return strings.ToLower(lobbies[i].name) < strings.ToLower(lobbies[j].name)
	})
	return lobbies
}

func (r *lobbyRegistry) Join(id uint64, username string) (*lobbyCommon, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	lobby, ok := r.lobbies[id]
	if !ok {
		return nil, errors.New("that lobby no longer exists")
	}
	lobby.m.Lock()
	defer lobby.m.Unlock()
	if lobby.isStarted || lobby.resuming {
		return nil, errors.New("that game has already started")
	}
	if len(lobby.players) >= 4 {
		return nil, errors.New("lobby full")
	}
	if _, exists := lobby.players[username]; exists {
		return nil, errors.New("username already taken in this lobby")
	}
	lobby.players[username] = &lobbyPlayer{}
	return lobby, nil
}

func (r *lobbyRegistry) Leave(lobby *lobbyCommon, username string) {
	var started []string
	// Registered first so logging runs after both deferred unlocks.
	defer func() { lobby.logGameStart(started) }()
	// Always acquire the registry lock before the lobby lock.
	r.m.Lock()
	defer r.m.Unlock()
	lobby.m.Lock()
	defer lobby.m.Unlock()
	delete(lobby.players, username)
	if len(lobby.players) == 0 {
		delete(r.lobbies, lobby.id)
	} else {
		started = lobby.startIfReady()
	}
}

func (l *lobbyCommon) toggleReady(username string) {
	var started []string
	defer func() { l.logGameStart(started) }()
	l.m.Lock()
	defer l.m.Unlock()
	if l.isStarted {
		return
	}
	player := l.players[username]
	if player == nil {
		return
	}
	player.isReady = !player.isReady
	started = l.startIfReady()
}

// Caller holds the lobby lock. Returns player names only when a game starts.
func (l *lobbyCommon) startIfReady() []string {
	if l.isStarted {
		return nil
	}
	if len(l.players) < 3 {
		return nil
	}
	for _, player := range l.players {
		if !player.isReady {
			return nil
		}
	}
	if l.resuming {
		if len(l.players) != len(l.seats) {
			return nil
		}
		l.isStarted = true
		names := make([]string, len(l.seats))
		for name, index := range l.seats {
			names[index] = name
		}
		return names
	}
	// Assign indices in the same pass that builds the game's player order.
	names := make([]string, 0, len(l.players))
	for name, player := range l.players {
		player.playerIndex = len(names)
		names = append(names, name)
	}
	l.game.StartOptions(names, false, false)
	l.isStarted = true
	return names
}

func (l *lobbyCommon) logGameStart(players []string) {
	if players != nil {
		log.Printf("game started: lobby=%q id=%d players=%q", l.name, l.id, players)
	}
}
