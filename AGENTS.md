# El Poblador

Read [README.md](README.md) before working in this repository. It covers the
project, running local games, controls, and SSH multiplayer. For deployment work,
also read [deploy/gcp/README.md](deploy/gcp/README.md).

Keep README.md focused on introducing the project and explaining how to use it;
it should not become a list of features. Most bug fixes and feature implementations
should not modify the README at all. Update it only when essential setup or usage
instructions need to change.

El Poblador is a terminal implementation of Settlers of Catan for 3–4 players,
built with Go and **Bubble Tea**. Lip Gloss provides terminal styling, Bubbles
provides UI components, and Wish hosts the multiplayer interface over SSH.

## Repository map

- `game/`: game state, rules, phase transitions, trading, and game layout.
- `board/`: hex board geometry, placement rules, resources, and board rendering.
- `main.go`: CLI commands (`new`, `load`, `serve`) and local/game Bubble Tea models.
- `serve.go`: SSH session lifecycle, navigation, and periodic refresh.
- `lobby.go` and `lobby_views.go`: shared lobby registry, readiness, and lobby UI.
- `cmd/testboard/` and `cmd/fullscreen/`: board and terminal display utilities.

## Game state and phases

The central interfaces and state live in `game/game.go`:

- **`Phase`** represents the turn holder's current interaction. It embeds
  `PhaseView` (`MoveCursor`, `HelpText() string`, `Menu() RichText`) and adds
  `BoardCursor` and `ConfirmTurn() (Phase, EnterMode)`. Implementations live in `game/phase_*.go`
  and cover setup, dice rolls, building, trading, the robber, and development cards.
- **`EnterMode`** controls the phase stack: `EnterPush` adds a nested interaction,
  `EnterReplace` replaces its top, and `EnterReset` replaces the whole stack.
  Returning a nil next phase leaves the stack unchanged. Cancel pops the top
  phase only when another phase remains underneath.
- **`SpectatorPhase`** represents a non-turn holder's interaction, including
  making and accepting trades. It embeds `PhaseView` and adds
  `ConfirmSpectator() (SpectatorPhase, EnterMode)`. Each player has a separate
  spectator stack; trade phases implement both phase interfaces.
- **`Game`** owns the board, players, dice, turn holder, development deck, action
  log, trade offers, and phase stacks. `MoveCursor`, `ConfirmAction`, and
  `CancelAction` route input by player perspective and lock game state;
  `Print` locks for reading. Phase methods execute inside these entry points.
  `NextTurn` clears spectator stacks and trade offers.
- **`phaseWithOptions`** (`game/phase_base.go`) supplies shared menu selection
  behavior. Its optional player ID distinguishes a spectator from the turn holder.
- **`Player`** (`game/player.go`) stores resources, hidden/played development
  cards, and available building pieces. Player IDs are indices into `Game.Players`.
- **`TradeOffer`** (`game/trade.go`) stores the offerer, target, resource maps,
  and status. Target `-1` means anyone; `board.ResourceInvalid` represents an
  unspecified resource in an ambiguous offer.

## Board and rendering

`board.Board` stores tiles by `TileCoord`, roads by `PathCoord`, and settlements
and city upgrades by `CrossCoord`. Building maps hold player IDs; cities are
upgrades of existing settlements. The board also stores the robber position and
player colors. Coordinate constructors and neighbor helpers in
`board/coordinates.go` encode the board's geometry; `PathCoord` connects two
crossings. `Tile`, `TerrainType`, and `ResourceType` live in `board/terrain.go`.

Menus and `Game.ActionLog` share `game.RichText` (`game/rich_text.go`), a slice of
`TextPart` values containing text and deferred styles. `String()` returns plain
text; `Render(renderer)` applies styles for the viewer. Keep `HelpText()` plain.
Do not pre-render ANSI strings into shared menus or logs, or switch the global
renderer for an SSH session. Each SSH session uses its own renderer from Wish's
`bubbletea.MakeRenderer`, passed through the game and board view options.

`board.BoardView` carries transient renderer, cursor, and emoji preferences.
`game.PrintOptions` adds terminal dimensions, player perspective, and layout
selection. A nil renderer uses the local terminal renderer; a nil `RequestPlayer`
follows the turn holder. Gob saves persist exported game state;
phase stacks and view state are not saved. `loadGameState` restores the dice-roll
phase after decoding.

Save-format stability and compatibility with older saves are deferred to 1.0.
Avoid adding migration or compatibility machinery during unrelated changes.

## Bubble Tea and multiplayer

`GameConn` is the active-game Bubble Tea model: `Update` translates messages into
game actions and `View` calls `Game.Print`. `LocalConn` adds perspective switching.
`SessionModel` owns each SSH connection's active model, terminal dimensions,
navigation, cleanup, and single refresh loop.

`lobbyRegistry` shares lobbies across sessions. Each `lobbyCommon` owns membership,
readiness, and one shared game; `lobbyPlayer.playerIndex` binds a username to its
game player. Nested locking follows registry → lobby. Game actions and rendering
run after releasing both locks. A lobby starts when all 3–4 members are ready;
joining an already started game is unsupported.
