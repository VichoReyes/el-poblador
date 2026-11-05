# El Poblador - Settlers of Catan TUI

[![builds.sr.ht status](https://builds.sr.ht/~vicho/el_poblador/commits/main/test.yml.svg)](https://builds.sr.ht/~vicho/el_poblador/commits/main/test.yml?)

A terminal-based implementation of Settlers of Catan in Go.

![game screenshot](screenshot.jpg)

## Installation

Download the latest release for your platform from [release builds](https://builds.sr.ht/~vicho/el_poblador/commits/main/release.yml). Alternatively, you can run `go install git.sr.ht/~vicho/el_poblador@latest`.

Another option is cloning the repo and building from [source](https://git.sr.ht/~vicho/el_poblador/refs). Or making an AUR package. Go wild.

## Running the Project

The examples use `go run main.go`, but if you have a compiled binary use that instead.

Run the main game with 3-4 players:

```bash
go run main.go new <player1> <player2> <player3> [player4]  # player names
```

Load a saved game:

```bash
go run main.go load <savefile>
```

**Controls:**
- Arrow keys: Move cursor
- Enter: Confirm action
- Esc: Cancel action (not always available)
- 1-4: Switch to specific player's perspective
- 0: Switch back to current turn holder's perspective
- Ctrl+C: Quit game without saving

## Online Multiplayer

There's a hosted version, so you only need to connect to see players.

```bash
$ ssh el-poblador.vichoreyes.cl -p 2222  # to see open lobbies
```

It's on GCP, as documented on
[the deployment guide](deploy/gcp/README.md).

Connections open a lobby browser. Use the arrow keys to select a lobby and
Enter to join, or press `n` to create a named lobby.
In a lobby, press Enter to signal you're ready and want the game to start (if the lobby has 3 players all ready, it won't wait for a 4th one).

Rejoining games already in
progress is not supported yet.

### Self host

Start a server with

```
$ go run . serve
2026/08/01 17:50:49 starting server at 127.0.0.1:2222
```

then players can connect with

```
$ ssh username@<host> -p 2222
```

Options for the server can be checked with

```
$ go run . serve -help
```

This feature is in its early stages and as such has a bunch of
[rough edges](https://todo.sr.ht/~vicho/el_poblador?search=label%3A%22ssh%22).
Do send patches or bug reports.

## License

EUPL v1.2
