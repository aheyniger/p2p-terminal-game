package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"p2p_game/internal/game"
	. "p2p_game/internal/misc"
	network "p2p_game/internal/network"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
)

func main() {
	var mu sync.Mutex
	// fmt.Println("Hello! This will be the main executable for the game, but right now is unimplemented!")

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run main.go <port> [join_ip:port]")
		return
	}
	outgoingConn := len(os.Args) > 2
	gameNet := connectToLobby(outgoingConn)

	state := &game.WorldState{
		Players: make(map[game.PlayerId]*game.Player),
	}

	wv := game.NewWorldView()
	defer wv.CloseWorldView()

	cx, cy := wv.GetViewCenter()

	//TODO: should we change the ID type? like to just the uuid.UUID type or an int?
	localPlayer := &game.Player{
		Id:    uuid.New().String(),
		Color: rand.Int32(),
		Pos:   game.Vec2{X: cx, Y: cy},
	}
	gameNet.NodePlayers["local"] = localPlayer.Id
	state.Players[localPlayer.Id] = localPlayer

	gameNet.BroadcastJoin(localPlayer.Id, localPlayer.Color)

	// NETWORK HOOK (incoming msgs)
	gameNet.OnPositionUpdate = func(id string, x, y int) {
		mu.Lock()
		defer mu.Unlock()

		state.ApplyRemoteUpdate(id, x, y)
	}

	wv.Ui.SetHeaderField(
		"Location",
		fmt.Sprintf("%v,%v", localPlayer.Pos.X, localPlayer.Pos.Y),
	)

	// INPUT HANDLER
	keyInputHandler := func(e *tcell.EventKey) bool {
		lastKey := ""

		mu.Lock()
		defer mu.Unlock()

		switch e.Key() {
		case tcell.KeyEscape:
			return true

		case tcell.KeyUp:
			lastKey = "UpArrow"
			state.MovePlayer(localPlayer.Id, 0, -1)

		case tcell.KeyDown:
			lastKey = "DownArrow"
			state.MovePlayer(localPlayer.Id, 0, 1)

		case tcell.KeyLeft:
			lastKey = "LeftArrow"
			state.MovePlayer(localPlayer.Id, -1, 0)

		case tcell.KeyRight:
			lastKey = "RightArrow"
			state.MovePlayer(localPlayer.Id, 1, 0)

		case tcell.KeyRune:
			switch e.Rune() {
			case 'r', 'R':
				w, h := wv.GetViewSize()
				state.MovePlayer(localPlayer.Id, w/2-localPlayer.Pos.X, h/2-localPlayer.Pos.Y)
			}
		}

		// broadcast AFTER state change
		p := state.Players[localPlayer.Id]
		gameNet.BroadcastPosition(p.Id, p.Pos.X, p.Pos.Y)

		wv.Ui.SetHeaderField("Last input", lastKey)
		wv.Ui.SetHeaderField("Location",
			fmt.Sprintf("%v,%v", p.Pos.X, p.Pos.Y),
		)

		w, h := wv.GetViewSize()
		wv.Ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", w, h))

		return false
	}

	renderShutdownCh := make(chan bool)
	go func() {
		wv.RenderLoop(state, keyInputHandler)
		renderShutdownCh <- true
		close(renderShutdownCh)
	}()
	playerLeaveCh := make(chan string)
	gameNet.PlayerLeaveCh = playerLeaveCh

	// seenMsg := make(map[string]bool)

	gameNet.OnMsg = func(msg []byte) {
		OnMsgReceived(gameNet, state, string(msg))
	}

	// Keep running + print members periodically
	// reader := bufio.NewReader(os.Stdin)

	// var msgId atomic.Uint64

	// for {
	// 	fmt.Print("Enter log: ")
	// 	input, _ := reader.ReadString('\n')

	// 	input = strings.TrimSpace(input)

	// 	if input != "" {
	// 		timestamp := time.Now().UnixNano()
	// 		broadcastMsg := fmt.Sprintf("%s|%d|%d|%s", gameNet.LocalName, msgId.Add(1), timestamp, input)
	// 		gameNet.Broadcast(broadcastMsg)
	// 	}
	// }
	select {
	case shutdown := <-renderShutdownCh:
		if shutdown {
			if err := gameNet.List.Leave(5 * time.Second); err != nil {
				fmt.Printf("failed to leave: %w", err)
			}
			if err := gameNet.List.Shutdown(); err != nil {
				fmt.Printf("failed to shutdown: %w", err)
			}
		}
		return
	case playerId := <-playerLeaveCh:
		fmt.Printf("%s", playerId)
		// delete(state.Players, playerId)
	} // block forever
}

func OnMsgReceived(gameNet *network.Network, gameState *game.WorldState, msg string) {
	parts := strings.Split(msg, network.Delim)
	// if len(parts) != 4 {
	// 	log.Println("Malformed message:", logLine)
	// 	return
	// }

	node := parts[0]
	// timestamp := parts[1]
	msgType := MustAtoi(parts[2])

	switch network.MsgType(msgType) {
	case network.JOIN:
		pId := parts[3]
		pColor := MustAtoi(parts[4])
		newPlayer := &game.Player{
			Id:    pId,
			Color: int32(pColor),
			Pos:   game.Vec2{X: -1, Y: -1},
		}

		gameState.Players[newPlayer.Id] = newPlayer
		gameNet.NodePlayers[node] = newPlayer.Id

	case network.POS_UPDATE:
		pId := parts[3]
		x := MustAtoi(parts[4])
		y := MustAtoi(parts[5])
		gameNet.OnPositionUpdate(pId, x, y)

	}

	// id := parts[1]
	// // timestamp := parts[2]
	// message := parts[3]

	// // Deduplicate using ID
	// if seenMsg[id] {
	// 	return
	// }
	// seenMsg[id] = true

	// // Clean display
	// formatted := fmt.Sprintf("[%s] %s", node, message)

	// fmt.Println(formatted)

	// // Append to file
	// f, err := os.OpenFile("shared.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	log.Println("File error:", err)
	// 	return
	// }
	// defer f.Close()

	// f.WriteString(logLine + "\n")
}

func connectToLobby(outgoing bool) *network.Network {
	port := MustAtoi(os.Args[1])

	nodeIP := os.Getenv("NODE_IP")
	if nodeIP == "" {
		nodeIP = fmt.Sprintf("%v", network.GetOutboundIP())
	}
	fmt.Println("Outbound IP:", nodeIP)

	machineName := fmt.Sprintf("%s:%d", nodeIP, port)

	gameNet, err := network.CreateNetwork(machineName, nodeIP, port)
	if err != nil {
		log.Fatalf("Error starting game network: %v", err)
	}

	// If join address provided → join cluster
	if outgoing {
		joinAddr := os.Args[2]
		n, err := gameNet.List.Join([]string{joinAddr})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Joined %d nodes\n", n)
	}

	fmt.Println("Node started:", machineName)

	return gameNet
}
