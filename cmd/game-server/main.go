package main

import (
	"encoding/json"
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

	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
)

func main() {
	var mu sync.Mutex

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run main.go <port> [join_ip:port]")
		return
	}
	outgoingConn := len(os.Args) > 2
	logCh := make(chan string, 256)
	gameNet := connectToLobby(outgoingConn, logCh)

	state := &game.WorldState{
		Players: make(map[game.PlayerId]*game.Player),
		Blocks:  make(map[string]*game.Block),
	}

	// Set up TCP State Sync Callbacks
	gameNet.GetLocalState = func() []byte {
		mu.Lock()
		defer mu.Unlock()

		// Serialize the current blocks map into JSON
		b, err := json.Marshal(state.Blocks)
		if err != nil {
			log.Println("Error marshaling local state:", err)
			return []byte{}
		}
		return b
	}

	gameNet.MergeState = func(buf []byte) {
		mu.Lock()
		defer mu.Unlock()

		var remoteBlocks map[string]*game.Block
		if err := json.Unmarshal(buf, &remoteBlocks); err != nil {
			log.Println("Error unmarshaling remote state:", err)
			return
		}

		// Merge the incoming blocks into the local state safely
		for id, block := range remoteBlocks {
			if _, exists := state.Blocks[id]; !exists {
				state.Blocks[id] = block
			}
		}
	}

	f, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("File error:", err)
		return
	}
	defer f.Close()

	wv := game.NewWorldView()
	defer wv.CloseWorldView()

	log.SetOutput(f)
	// os.Stdout = f
	// os.Stderr = f

	cx, cy := wv.GetViewCenter()

	//TODO: should we change the ID type? like to just the uuid.UUID type or an int?
	localPlayer := &game.Player{
		Id:    uuid.New().String(),
		Color: game.GetColorFromID(uuid.New().String()),
		Pos:   game.Vec2{X: cx, Y: cy},
	}
	gameNet.NodePlayers["local"] = localPlayer.Id
	state.Players[localPlayer.Id] = localPlayer
	leaveEventCh := make(chan network.LeaveEvent)
	gameNet.LeaveEventCh = leaveEventCh

	gameNet.BroadcastJoin(localPlayer.Id, localPlayer.Color)

	// Wait for cluster to acknowledge us before broadcasting our new blocks
	if outgoingConn {
		// Wait until memberlist actually registers at least 1 other node
		for len(gameNet.List.Members()) < 2 {
			time.Sleep(100 * time.Millisecond)
		}
		// Add a tiny buffer to allow remote nodes to stabilize their UDP pipelines
		time.Sleep(200 * time.Millisecond)
	}
	//spawn blocks
	//one node spawns blocks in random locations upon another node joining
	// log.Printf("DEBUG compare: local='%q' node='%q'\n", gameNet.LocalName, node)
	// log.Printf("LEN local=%d node=%d\n", len(gameNet.LocalName), len(node))
	// if gameNet.LocalName == node {
	// log.Printf("gamenet.localName == node success'\n")

	numBlocks := 1 //1 blocks per player. change if needed
	for i := 0; i < numBlocks; i++ {
		blockID := uuid.New().String()

		block := &game.Block{
			ID: blockID,
			Pos: game.Vec2{ //plae block in random location
				X: rand.IntN(50),
				Y: rand.IntN(20),
			},
			HeldBy:    "", //held by no one
			OwnerNode: gameNet.LocalName,
		}

		// fmt.Println("SPAWN:", block.ID)
		// fmt.Println("RECV BLOCK:", blockID)
		// fmt.Println("TOTAL BLOCKS:", len(gameState.Blocks))
		log.Printf("SPAWN: %s'\n", block.ID)
		log.Printf("recv: %s'\n", blockID)
		// log.Printf("LEN local=%d node=%d\n", len(gameNet.LocalName), len(node))

		state.Blocks[block.ID] = block

		//broadcast to other nodes to spawn blocks:
		gameNet.BroadcastBlockSpawn(block.ID, block.Pos.X, block.Pos.Y, block.OwnerNode)
	}
	// } else {
	// 	log.Printf("did not enter if statement\n")
	// }

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
			case 'm', 'M':
				convergenceTime := gameNet.TestBroadcastAndMeasure()
				log.Printf("Convergence time measured at %dms\n", convergenceTime.Milliseconds())
			case 'r', 'R':
				w, h := wv.GetViewSize()
				state.MovePlayer(localPlayer.Id, w/2-localPlayer.Pos.X, h/2-localPlayer.Pos.Y)
			case ' ':
				//fetch latest player state:
				livePlayer := state.Players[localPlayer.Id]
				// attempt block grab
				log.Printf("DEBUG [INPUT]: Spacebar pressed at player pos X:%d Y:%d\n", livePlayer.Pos.X, livePlayer.Pos.Y)
				if livePlayer.HeldBlock != nil { //if currently holding block, drop it
					log.Printf("DEBUG [INPUT]: dropping BLOCK X:%d Y:%d\n", livePlayer.Pos.X, livePlayer.Pos.Y)
					b := livePlayer.HeldBlock
					gameNet.BroadcastDropRequest(b.ID, livePlayer.Id, livePlayer.Pos.X, livePlayer.Pos.Y, b.OwnerNode)
					break
				}
				for _, b := range state.Blocks { //find block at the players position
					log.Printf("DEBUG [INPUT]: Checking block %s at X:%d Y:%d\n", b.ID, b.Pos.X, b.Pos.Y)
					if b.Pos.X == livePlayer.Pos.X && b.Pos.Y == livePlayer.Pos.Y {
						log.Printf("DEBUG [INPUT]: Hitbox matched! Block is held by: '%s'. Requesting from owner: %s\n", b.HeldBy, b.OwnerNode)
						if b.HeldBy == "" {
							// don't assign it yet! ask the owner for permission.
							gameNet.BroadcastGrabRequest(b.ID, livePlayer.Id, b.OwnerNode)
						}
						break
					}
				}
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
	// playerLeaveCh := make(chan string)
	// gameNet.PlayerLeaveCh = playerLeaveCh

	// seenMsg := make(map[string]bool)

	gameNet.OnMsg = func(msg []byte) {
		OnMsgReceived(gameNet, state, string(msg))
	}

	for {
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

		case line := <-logCh:
			log.Println(line)
			wv.SetLogLine(line)

		case event := <-leaveEventCh:
			mu.Lock()

			// 1. remove the player
			delete(state.Players, event.PlayerID)

			// 2. figure out who takes over the blocks
			newOwner := getNextOwner(gameNet, event.NodeName)

			// 3. reassign orphaned blocks
			for _, b := range state.Blocks {
				if b.OwnerNode == event.NodeName {
					b.OwnerNode = newOwner

					// safety check: if the player who left was currently holding the block, drop it
					if b.HeldBy == event.PlayerID {
						b.HeldBy = ""
					}
				}
			}

			mu.Unlock()
		}

	}

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
		// pColor := MustAtoi(parts[4])
		newPlayer := &game.Player{
			Id:    pId,
			Color: game.GetColorFromID(pId),
			Pos:   game.Vec2{X: -1, Y: -1},
		}

		gameState.Players[newPlayer.Id] = newPlayer
		gameNet.NodePlayers[node] = newPlayer.Id

	case network.POS_UPDATE:
		pId := parts[3]
		x := MustAtoi(parts[4])
		y := MustAtoi(parts[5])

		//if first time seeig player, make them
		if _, exists := gameState.Players[pId]; !exists {
			gameState.Players[pId] = &game.Player{
				Id:  pId,
				Pos: game.Vec2{X: x, Y: y},
				// Assign their permanent color instantly without asking the network
				Color: game.GetColorFromID(pId),
			}
		} else {
			// They already exist, just update coordinates
			gameState.Players[pId].Pos.X = x
			gameState.Players[pId].Pos.Y = y
		}

		gameNet.OnPositionUpdate(pId, x, y)

	case network.GRAB_REQ:
		blockID := parts[3]
		playerID := parts[4]
		owner := parts[5]

		log.Printf("DEBUG [GRAB_REQ]: Received req for block %s by player %s. Target Owner: %s, My Name: %s\n", blockID, playerID, owner, gameNet.LocalName)

		// Only owner processes
		if owner != gameNet.LocalName {
			return
		}

		// b, exists := gameState.FindBlockByID(blockID)
		b := gameState.FindBlockByID(blockID)
		if b == nil {
			log.Printf("DEBUG [GRAB_REQ]: Block %s not found in my state!\n", blockID)
			return
		}

		success := false

		if b.HeldBy == "" {
			b.HeldBy = playerID
			success = true
			log.Printf("DEBUG [GRAB_REQ]: Approved grab! Assigning to %s\n", playerID)

			//apply changes locally, since memberlist only broadcasts to others
			if p, pExists := gameState.Players[playerID]; pExists {
				p.HeldBlock = b
				log.Println("DEBUG [GRAB_REQ]: Applied HeldBlock state locally for the owner node.")
			}
		}

		// broadcast result
		gameNet.BroadcastGrabResult(blockID, playerID, success, b.OwnerNode)

	case network.GRAB_RES:
		log.Printf("DEBUG [GRAB_RES]: Received message! Raw parts: %v\n", parts)

		blockID := parts[3]
		playerID := parts[4]
		success := MustAtoi(parts[5]) == 1

		log.Printf("DEBUG [GRAB_RES]: Parsed - Block: %s, Player: %s, Success: %v\n", blockID, playerID, success)

		if !success {
			log.Println("DEBUG [GRAB_RES]: Success was false, returning early.")
			return
		}

		b, exists := gameState.Blocks[blockID]
		if !exists {
			log.Printf("DEBUG [GRAB_RES]: Block %s not found in gameState!\n", blockID)
			return
		}

		b.HeldBy = playerID
		log.Printf("DEBUG [GRAB_RES]: Set block %s HeldBy to %s\n", blockID, playerID)

		if p, pExists := gameState.Players[playerID]; pExists {
			p.HeldBlock = b
			log.Printf("DEBUG [GRAB_RES]: SUCCESS! Attached block to player %s's HeldBlock field!\n", playerID)
		} else {
			log.Printf("DEBUG [GRAB_RES]: ERROR - Player %s not found in gameState.Players!\n", playerID)
		}

	case network.DROP_REQ:
		blockID := parts[3]
		playerID := parts[4]
		dropX := MustAtoi(parts[5]) // Assuming you have a MustAtoi helper, or use strconv.Atoi
		dropY := MustAtoi(parts[6])
		owner := parts[7]

		if owner != gameNet.LocalName {
			return
		}

		b, exists := gameState.Blocks[blockID]
		if !exists {
			return
		}

		isOccupied := false //check if we can place block there, or if theres already a block

		for _, otherBlock := range gameState.Blocks {
			if otherBlock.ID == blockID {
				continue
			}
			if otherBlock.Pos.X == dropX && otherBlock.Pos.Y == dropY && otherBlock.HeldBy == "" {
				isOccupied = true
				break
			}
		}

		success := false
		// Validate that the requester actually owns the block right now and that the space is free
		if !isOccupied && b.HeldBy == playerID {
			b.HeldBy = ""
			b.Pos.X = dropX
			b.Pos.Y = dropY
			success = true

			if p, pExists := gameState.Players[playerID]; pExists {
				p.HeldBlock = nil
				log.Println("DEBUG [DROP_REQ]: Applied HeldBlock state locally for the owner node.")
			}
		}

		gameNet.BroadcastDropResult(blockID, playerID, dropX, dropY, success)

	case network.DROP_RES:
		blockID := parts[3]
		playerID := parts[4]
		dropX := MustAtoi(parts[5])
		dropY := MustAtoi(parts[6])
		success := parts[7] == "1"

		if !success {
			return
		}

		b, bExists := gameState.Blocks[blockID]
		if !bExists {
			return
		}

		// 1. Unassign the block and update its physical coordinates
		b.HeldBy = ""
		b.Pos.X = dropX
		b.Pos.Y = dropY

		// 2. Clear the block from the player, making them shrink back to 1x1
		if p, pExists := gameState.Players[playerID]; pExists {
			// Double check they are holding THIS block to prevent race conditions
			if p.HeldBlock != nil && p.HeldBlock.ID == blockID {
				p.HeldBlock = nil
			}
		}

	case network.BLOCK_SPAWN:
		blockID := parts[3]
		x := MustAtoi(parts[4])
		y := MustAtoi(parts[5])
		owner := parts[6]

		if _, exists := gameState.Blocks[blockID]; exists {
			return //so we dont overwrite
		}

		gameState.Blocks[blockID] = &game.Block{
			ID:        blockID,
			Pos:       game.Vec2{X: x, Y: y},
			HeldBy:    "",
			OwnerNode: owner,
		}

	case network.STATE_SYNC:
		blockID := parts[3]
		x := MustAtoi(parts[4])
		y := MustAtoi(parts[5])
		owner := parts[6]

		// Avoid duplicates
		if _, exists := gameState.Blocks[blockID]; exists {
			return
		}

		gameState.Blocks[blockID] = &game.Block{
			ID:        blockID,
			Pos:       game.Vec2{X: x, Y: y},
			HeldBy:    "",
			OwnerNode: owner,
		}

	case network.TEST_GOSSIP:
		gameNet.SendTestGossipAck(node, parts[3])

	case network.TEST_GOSSIP_ACK:
		if ch, ok := gameNet.PendingAcks.Load(parts[3]); ok {
			ch.(chan network.NodeTime) <- network.NodeTime{AckTime: time.Now(), NodeName: node}
		}
	}

}

func getNextOwner(gameNet *network.Network, leavingNode string) string {
	var activeNodes []string

	for _, m := range gameNet.List.Members() {
		// exclude the node that is actively leaving
		if m.Name != leavingNode {
			activeNodes = append(activeNodes, m.Name)
		}
	}

	// if you are the last node standing, you own everything
	if len(activeNodes) == 0 {
		return gameNet.LocalName
	}

	// alphabetically sort the remaining nodes to ensure deterministic selection
	sort.Strings(activeNodes)
	return activeNodes[0]
}

func connectToLobby(outgoing bool, logCh chan string) *network.Network {
	port := MustAtoi(os.Args[1])

	nodeIP := os.Getenv("NODE_IP")
	if nodeIP == "" {
		nodeIP = fmt.Sprintf("%v", network.GetOutboundIP())
	}
	fmt.Println("Outbound IP:", nodeIP)

	machineName := fmt.Sprintf("%s:%d", nodeIP, port)

	gameNet, err := network.CreateNetwork(machineName, nodeIP, port, logCh)
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
