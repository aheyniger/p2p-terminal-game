# p2p-terminal-game
## User Instructions
### (Optional) Compiling
If you wish to use a binary of the game, run `make build` to compile the project, and the binary will appear in `/build` as `game-server`. Compiling is optional and the next section covers how to run it with or without a binary.

### Running The Game
The game can be run with either `go run cmd/game-server/main.go`, or if you compiled it, `./build/game-server`. These can be used interchangeably.

Usage is `./build/game-server <local port> [hostname:port]`

**Arguments:**
- `<local port>`: The local port to bind and advertise for other nodes to join. **This port is used for both UDP and TCP**
    - If running on Lehigh's Sunlab, only port 4041 works for UDP, and so multiple nodes cannot be run on the same machine.
- `[hostname:port]`: If joining another session, this is the hostname (or IP address) and TCP/UDP port to join. **This can be any host that is already in the session.** If you are starting a session, this should be omitted. An example argument would be `eris:4041`.

### Playing The Game
When the game is successfully started, your terminal will switch to the game world TUI. 
- Use arrow keys to move your character around
- Press `r` to reset your position to its beginning location
- Press the spacebar while standing on a block to pick it up
    - If already holding a block, press spacebar anywhere to drop it
- Press Escape to exit the game

## Development Instructions & Notes
### Aliases
In `scripts/`, there is a Bash script called `aliases.sh`, which contains aliases that might be helpful. Get/source the aliases from the root directory with `source scripts/aliases.sh`

**Feel free to add more aliases; they are there for ease of development**

### Building and running
There is a `Makefile` to make building and running a bit easier

- To run directly from source, use `make run CMD=<main package directory in cmd/>`, or omit `CMD` from that command to just run the actual game at `cmd/game-server`
    - For example, to run the test ui, run `make run CMD=test_ui` (or the alias `runui`)
    - Running `make run` is equivalent to running `make run CMD=game-server`
- To build an executable (which will be stored in `build/`), use the same format as the above `make run`, but instead using `make build`


### File structure
- Executable binaries are put into `build/`
- All main packages to define a `func main()` should go in their own folder in `cmd/`
- All code not in a `main()` should go into some directory in `internal/`
- Scripts go into `scripts/`
