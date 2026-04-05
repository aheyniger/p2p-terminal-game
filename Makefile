CMD ?= game-server

# EXEC ?= mygame
MAIN_PKG=./cmd/$(CMD)

build:
	go build -o ./build/$(CMD) $(MAIN_PKG)

run:
	go run $(MAIN_PKG)


.PHONY: build run clean test fmt vet tidy