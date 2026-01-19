# El Poblador - Settlers of Catan TUI

[![builds.sr.ht status](https://builds.sr.ht/~vicho/el_poblador/commits/main/test.yml.svg)](https://builds.sr.ht/~vicho/el_poblador/commits/main/test.yml?)

A terminal-based implementation of Settlers of Catan in Go.

![game screenshot](screenshot.png)

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
- q/Ctrl+C: Quit game  (to be removed)

## Online Multiplayer

Online multiplayer is one of the [missing features](https://todo.sr.ht/~vicho/el_poblador/).
It would be cool to have, though.

## License

EUPL v1.2, in spanish
