package tui

import (
	"fmt"
	"reflect"
	"testing"
)

func makeTestLobby(t *testing.T, registry *lobbyRegistry, name string, players ...string) *lobbyCommon {
	t.Helper()
	lobby, err := registry.Create(name, players[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range players[1:] {
		if _, err := registry.Join(lobby.id, name); err != nil {
			t.Fatal(err)
		}
	}
	return lobby
}

// Only call after the workers have completed, while no game actions are running.
func assertStartedPlayers(t *testing.T, lobby *lobbyCommon, names ...string) {
	t.Helper()
	lobby.m.RLock()
	defer lobby.m.RUnlock()
	if !lobby.isStarted {
		t.Fatal("game did not start")
	}
	if len(lobby.game.Players) != len(names) {
		t.Fatalf("game has %d players, want %d", len(lobby.game.Players), len(names))
	}
	for _, name := range names {
		player := lobby.players[name]
		if player == nil {
			t.Errorf("missing player %s", name)
			continue
		}
		index := player.playerIndex
		if index < 0 || index >= len(lobby.game.Players) {
			t.Errorf("invalid player index %d for %s", index, name)
			continue
		}
		if got := lobby.game.Players[index].Name; got != name {
			t.Errorf("%s was assigned to %s", name, got)
		}
	}
}

func TestLobbyReadinessRequiresThreeOrFourReadyPlayers(t *testing.T) {
	for count := 1; count <= 4; count++ {
		t.Run(fmt.Sprintf("%d_players", count), func(t *testing.T) {
			registry := newLobbyRegistry()
			names := []string{"alice", "bob", "carol", "dave"}[:count]
			lobby := makeTestLobby(t, registry, "Ready", names...)
			for i, name := range names {
				lobby.toggleReady(name)
				wantStarted := count >= 3 && i == count-1
				if got := registry.List()[0].started; got != wantStarted {
					t.Fatalf("after %d ready players: started=%v, want %v", i+1, got, wantStarted)
				}
			}
			if count >= 3 {
				assertStartedPlayers(t, lobby, names...)
			}
		})
	}
}

func TestLobbyUnreadyAndNewMemberDelayStart(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Wait", "alice", "bob", "carol")
	lobby.toggleReady("alice")
	lobby.toggleReady("alice") // Withdraw readiness before the others are ready.
	lobby.toggleReady("bob")
	lobby.toggleReady("carol")
	if registry.List()[0].started {
		t.Fatal("game started after alice withdrew readiness")
	}
	if _, err := registry.Join(lobby.id, "dave"); err != nil {
		t.Fatal(err)
	}
	lobby.toggleReady("alice")
	if registry.List()[0].started {
		t.Fatal("new player inherited a ready seat")
	}
	lobby.toggleReady("dave")
	assertStartedPlayers(t, lobby, "alice", "bob", "carol", "dave")
}

func TestLobbyRejectsDuplicateUsernameWithoutChangingMembership(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Duplicates", "alice")
	lobby.toggleReady("alice")
	if _, err := registry.Join(lobby.id, "alice"); err == nil {
		t.Fatal("accepted a duplicate username")
	}
	if registry.List()[0].players != 1 || !lobby.players["alice"].isReady {
		t.Fatal("rejected join changed the existing player's seat or readiness")
	}
	// The same username is allowed in another independent lobby.
	makeTestLobby(t, registry, "Other", "alice")
}

func TestLobbyLeavingCreatorKeepsRemainingPlayers(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Survives", "alice", "bob", "carol")
	lobby.toggleReady("bob")
	lobby.toggleReady("carol")
	registry.Leave(lobby, "alice")
	rows := registry.List()
	if len(rows) != 1 || rows[0].players != 2 || rows[0].started {
		t.Fatalf("creator leaving produced invalid lobby state: %+v", rows)
	}
	if _, err := registry.Join(lobby.id, "dave"); err != nil {
		t.Fatal(err)
	}
	lobby.toggleReady("dave")
	assertStartedPlayers(t, lobby, "bob", "carol", "dave")
}

func TestLobbyNameReuseDoesNotReviveStaleIDs(t *testing.T) {
	registry := newLobbyRegistry()
	old := makeTestLobby(t, registry, "Reusable", "alice")
	registry.Leave(old, "alice")
	current := makeTestLobby(t, registry, "Reusable", "alice")
	if current.id == old.id {
		t.Fatal("replacement lobby reused a stale ID")
	}
	if _, err := registry.Join(old.id, "bob"); err == nil {
		t.Fatal("stale browser entry joined a replacement lobby")
	}
	registry.Leave(old, "alice") // Repeated stale cleanup must not delete the replacement.
	if rows := registry.List(); len(rows) != 1 || rows[0].id != current.id {
		t.Fatalf("stale cleanup removed the replacement: %+v", rows)
	}
	if _, err := registry.Join(current.id, "bob"); err != nil {
		t.Fatal(err)
	}
}

func TestLobbyRejoiningWaitingRoomResetsReadiness(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Rejoin", "alice", "bob")
	lobby.toggleReady("bob")
	registry.Leave(lobby, "bob")
	if _, err := registry.Join(lobby.id, "bob"); err != nil {
		t.Fatal(err)
	}
	if lobby.players["bob"].isReady {
		t.Fatal("rejoining restored readiness from the previous session")
	}
}

func TestLobbyReadinessCannotRestartRunningGame(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Once", "alice", "bob", "carol")
	for _, name := range []string{"alice", "bob", "carol"} {
		lobby.toggleReady(name)
	}
	assertStartedPlayers(t, lobby, "alice", "bob", "carol")
	lobby.game.ConfirmAction(nil) // The first player's settlement consumes one piece.
	pieces := lobby.game.Players[0].SettlementsAvailable
	for i := 0; i < 8; i++ {
		for _, name := range []string{"alice", "bob", "carol"} {
			lobby.toggleReady(name)
		}
	}
	if lobby.game.Players[0].SettlementsAvailable != pieces {
		t.Fatal("readiness messages reset an active game")
	}
	assertStartedPlayers(t, lobby, "alice", "bob", "carol")
}

func TestLobbyDisconnectKeepsRunningGamePlayerIndices(t *testing.T) {
	registry := newLobbyRegistry()
	lobby := makeTestLobby(t, registry, "Fixed seats", "alice", "bob", "carol", "dave")
	for name := range lobby.players {
		lobby.toggleReady(name)
	}
	indices := make(map[string]int)
	for name, player := range lobby.players {
		indices[name] = player.playerIndex
	}
	registry.Leave(lobby, "alice")
	registry.Leave(lobby, "bob")
	if rows := registry.List(); len(rows) != 1 || rows[0].players != 2 || !rows[0].started {
		t.Fatalf("disconnect changed game status: %+v", rows)
	}
	for _, name := range []string{"carol", "dave"} {
		if lobby.players[name].playerIndex != indices[name] || lobby.game.Players[indices[name]].Name != name {
			t.Errorf("disconnect reassigned %s", name)
		}
	}
	if len(lobby.game.Players) != 4 {
		t.Fatal("disconnect changed the game's original roster")
	}
	if _, err := registry.Join(lobby.id, "alice"); err == nil {
		t.Fatal("accepted a reconnect into a running game")
	}
	registry.Leave(lobby, "carol")
	registry.Leave(lobby, "dave")
	if len(registry.List()) != 0 {
		t.Fatal("last disconnect retained a running lobby")
	}
}

func TestLobbyListReturnsSortedIndependentSnapshots(t *testing.T) {
	registry := newLobbyRegistry()
	makeTestLobby(t, registry, "Zulu", "host")
	alpha := makeTestLobby(t, registry, "alpha", "host")
	before := registry.List()
	if got := []string{before[0].name, before[1].name}; !reflect.DeepEqual(got, []string{"alpha", "Zulu"}) {
		t.Fatalf("unexpected browser order: %v", got)
	}
	registry.Join(alpha.id, "guest")
	if before[0].players != 1 {
		t.Fatal("joining mutated an already returned snapshot")
	}
	before[0].name = "Changed by caller"
	if after := registry.List(); after[0].name != "alpha" || after[0].players != 2 {
		t.Fatalf("snapshot changed shared state or refresh was stale: %+v", after)
	}
}
