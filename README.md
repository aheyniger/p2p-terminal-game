# p2p-terminal-game


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


### Run main.go
Open 3 terminals. In src:
Terminal 1 run: go run main.go 8000
Terminal 2: go run main.go 8001 127.0.0.1:8000
Terminal 3: go run main.go 8002 127.0.0.1:8000

Enter messages to be broadcasted to other nodes. Other nodes should display "Received: <msg>".Messages are also saved in shared.log. Currently there are duplicate messages because theyre broadcasted via gossip protocol but duplicates can be removed later. 