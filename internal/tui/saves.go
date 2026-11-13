package tui

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/VichoReyes/el-poblador/game"
)

const saveLifetime = 30 * 24 * time.Hour
const maxSaveBytes = 1 << 20
const maxStoreBytes = 64 << 20

type saveStore struct {
	mu  sync.Mutex
	dir string
}
type savedLobby struct {
	Name string
	Game *game.Game
}

func normalizeCode(code string) (string, error) {
	code = strings.ToUpper(code)
	if len(code) != 16 {
		return "", errors.New("invalid save code")
	}
	if _, err := hex.DecodeString(code); err != nil {
		return "", errors.New("invalid save code")
	}
	return code, nil
}

// prune also accounts for existing snapshots before admitting another write.
func (s *saveStore) prune() (int64, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, entry := range entries {
		// Recover temporary snapshots left behind by an interrupted write.
		if strings.HasPrefix(entry.Name(), ".save-") {
			if err := os.Remove(filepath.Join(s.dir, entry.Name())); err != nil {
				return 0, err
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".gob") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		if time.Since(info.ModTime()) >= saveLifetime {
			if err := os.Remove(filepath.Join(s.dir, entry.Name())); err != nil {
				return 0, err
			}
		} else {
			total += info.Size()
		}
	}
	return total, nil
}

func (s *saveStore) write(code, name string, g *game.Game) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return "", err
	}
	total, err := s.prune()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(savedLobby{name, g}); err != nil {
		return "", err
	}
	if buf.Len() > maxSaveBytes {
		return "", errors.New("save exceeds 1 MiB limit")
	}
	path := filepath.Join(s.dir, code+".gob")
	var previous int64
	if info, err := os.Stat(path); err == nil {
		previous = info.Size()
	}
	if total-previous+int64(buf.Len()) > maxStoreBytes {
		return "", errors.New("save storage is full; game has not been paused")
	}
	f, err := os.CreateTemp(s.dir, ".save-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(buf.Bytes()); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return "", err
	}
	return fmt.Sprintf("Game saved. Resume code: %s\nReconnect with the same username:\nssh -t -p <port> <username>@<host> resume %s\nSave expires in 30 days. Ctrl+C to disconnect.", code, code), nil
}

func (r *lobbyRegistry) configureSave(l *lobbyCommon) {
	l.game.SetSaveHandler(func(g *game.Game) (string, error) {
		if l.code == "" {
			var b [8]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			l.code = strings.ToUpper(hex.EncodeToString(b[:]))
		}
		return r.saves.write(l.code, l.name, g)
	})
}

func (r *lobbyRegistry) Resume(code, username string) (*lobbyCommon, error) {
	code, err := normalizeCode(code)
	if err != nil {
		return nil, err
	}
	r.m.Lock()
	defer r.m.Unlock()
	for _, l := range r.lobbies {
		// code is assigned while holding the game lock, not the lobby lock.
		// Only resumed lobbies participate here; their code is immutable.
		if l.resuming && l.code == code {
			return joinSaved(l, username)
		}
	}
	r.saves.mu.Lock()
	defer r.saves.mu.Unlock()
	path := filepath.Join(r.saves.dir, code+".gob")
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) >= saveLifetime {
		return nil, errors.New("save code not found or expired")
	}
	if info.Size() > maxSaveBytes {
		return nil, errors.New("save exceeds size limit")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := &lobbyCommon{code: code, resuming: true, players: make(map[string]*lobbyPlayer)}
	saved := savedLobby{Game: &l.game}
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&saved); err != nil {
		return nil, err
	}
	l.name = saved.Name
	l.seats = make(map[string]int)
	for i, p := range l.game.Players {
		l.seats[p.Name] = i
	}
	if len(l.seats) < 3 || len(l.seats) > 4 {
		return nil, errors.New("invalid saved players")
	}
	l.game.SetPhase(game.PhaseDiceRoll(&l.game))
	if _, err := joinSaved(l, username); err != nil {
		return nil, err
	}
	r.nextID++
	l.id = r.nextID
	r.configureSave(l)
	r.lobbies[l.id] = l
	return l, nil
}

func joinSaved(l *lobbyCommon, username string) (*lobbyCommon, error) {
	l.m.Lock()
	defer l.m.Unlock()
	index, ok := l.seats[username]
	if !ok {
		return nil, errors.New("use your original player username to resume")
	}
	if l.isStarted {
		return nil, errors.New("that game has already resumed")
	}
	if _, ok := l.players[username]; ok {
		return nil, errors.New("that player is already connected")
	}
	l.players[username] = &lobbyPlayer{playerIndex: index}
	return l, nil
}
