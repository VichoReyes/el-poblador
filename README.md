# El Poblador - Settlers of Catan TUI

[![CI](https://github.com/VichoReyes/el-poblador/actions/workflows/ci.yml/badge.svg)](https://github.com/VichoReyes/el-poblador/actions/workflows/ci.yml)

A terminal-based implementation of Settlers of Catan in Go.

![game screenshot](screenshot.jpg)

**TLDR:** To play online, run `ssh catan.vichoreyes.cl -p 2222`

## Installation

Download the latest release for your platform from [GitHub Releases](https://github.com/VichoReyes/el-poblador/releases/latest). Alternatively, you can run `go install github.com/VichoReyes/el-poblador@main`.

Another option is cloning the repo and building from [source](https://github.com/VichoReyes/el-poblador). Or making an AUR package. Go wild.

## Running the Project

The examples use `go run .`, but if you have a compiled binary use that instead.

Run the main game with 3-4 players:

```bash
go run . new <player1> <player2> <player3> [player4]  # player names
```

Load a saved game:

```bash
go run . load <savefile>
```

**Controls:**
- Arrow keys: Move cursor
- Enter: Confirm action
- Esc: Cancel action (not always available)
- e: **toggle emojis** on/off, because they may break alignment on some terminal emulators.
- 1-4: Switch to specific player's perspective (only local play)
- 0: Switch back to current turn holder's perspective (only local play)
- Ctrl+C: Quit game without saving

## Online Multiplayer

There's a hosted version, so you only need to connect to see players.

```bash
$ ssh catan.vichoreyes.cl -p 2222  # to see open lobbies
```

It's on GCP, as documented on
[the deployment guide](deploy/gcp/README.md).

You can save and resume an online game. Resuming is done like so:
```bash
ssh -t -p 2222 yourname@catan.vichoreyes.cl resume YOUR_SAVE_CODE
```

Note: saved games are deleted after 30 days to save space.
Plus, I haven't yet committed to staying backwards-compatible with old game saves.

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

## License

EUPL v1.2
