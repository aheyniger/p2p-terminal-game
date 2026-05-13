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
	"hash/fnv"

	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
)

func main() {
	var mu sync.Mutex
	var danceCancel chan struct{}
	

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
		Pending: make(map[string]*game.PendingRequest),
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

	hostname, _ := os.Hostname()
	f, err := os.OpenFile(hostname+".debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	// Wait for cluster to acknowledge before broadcasting our new blocks
	if outgoingConn {
		// Wait until memberlist actually registers at least 1 other node
		for len(gameNet.List.Members()) < 2 {
			time.Sleep(100 * time.Millisecond)
		}
		// Add a buffer to allow remote nodes to stabilize their UDP pipelines
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
			HeldBy:    "",                //held by no one
			OwnerNode: gameNet.LocalName, //by default, original owner is the player that joined
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

	// Network hook (incoming msgs)
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
			case 'd', 'D':
				if danceCancel != nil {
					close(danceCancel)
					lastKey = "Dance STOP"
				} else {
					danceCancel = make(chan struct{})
					lastKey = "Dance START"

					go func(cancel chan struct{}) {
						ticker := time.NewTicker(200 * time.Millisecond) //change to 50 for faster dancing and more broadcasts (more load)
						defer ticker.Stop()
						for {
							select {
							case <-cancel:
								return
							case <-ticker.C:
								dx, dy := 0, 0
								switch rand.IntN(4) {
								case 0:
									dy = -1
								case 1:
									dy = 1
								case 2:
									dx = -1
								case 3:
									dx = 1
								}
								mu.Lock() //todo: why?

								state.MovePlayer(localPlayer.Id, dx, dy)
								p := state.Players[localPlayer.Id]

								gameNet.BroadcastPosition(p.Id, p.Pos.X, p.Pos.Y)
								wv.Ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", p.Pos.X, p.Pos.Y))

								mu.Unlock()
							}
						}
					}(danceCancel)
				}
			case 'm', 'M':
				convergenceTime := gameNet.TestBroadcastAndMeasure()
				log.Printf("Convergence time measured at %dms\n", convergenceTime.Milliseconds())
			case 'r', 'R':
				w, h := wv.GetViewSize()
				state.MovePlayer(localPlayer.Id, w/2-localPlayer.Pos.X, h/2-localPlayer.Pos.Y)
			case ' ':
				//generate req id
				reqID := uuid.NewString()
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
							
							var should_node_fail bool = false

							log.Printf(
								"[PERF STORE] req=%s",
								reqID,
							)

							//for measuring latency of grab requests to results
							network.PendingTimings.Store(reqID, network.RequestTiming{
								Start: time.Now(),
								Type:  "grab",
								NodeFail: should_node_fail,
							})

							state.Pending[reqID] = &game.PendingRequest{
								RequestID: reqID,
								Type:      game.PendingGrab,
								BlockID:   b.ID,
								PlayerID:  livePlayer.Id,
								OwnerNode: b.OwnerNode,
								CreatedAt: time.Now(),
							}
							gameNet.BroadcastPendingRequest(reqID, b.ID, string(game.PendingGrab), livePlayer.Id, b.OwnerNode, 0, 0)

							gameNet.BroadcastGrabRequest(b.ID, livePlayer.Id, b.OwnerNode, reqID)


							go func(targetBlockID, reqID, reqPlayerID string) {
							for i := 1; i <= 20; i++ {
								time.Sleep(1 * time.Second)
								
								 
								mu.Lock()
								p, pExists := state.Players[reqPlayerID]
								updatedBlock, bExists := state.Blocks[targetBlockID]
								
								// did we successfully grab it
								if pExists && p.HeldBlock != nil && p.HeldBlock.ID == targetBlockID {
									mu.Unlock()
									return // Success, exit the retry loop.
								}
								
								// is the block still available?
								if bExists && updatedBlock.HeldBy == "" {
									log.Printf("DEBUG [RETRY %d/10]: Grab %s pending. Current owner is: %s\n", i, reqID, updatedBlock.OwnerNode)
									
									// Broadcast again to whoever the current owner is
									gameNet.BroadcastGrabRequest(updatedBlock.ID, reqPlayerID, updatedBlock.OwnerNode, reqID)
								} else if bExists && updatedBlock.HeldBy != "" && updatedBlock.HeldBy != reqPlayerID {
									// someone else grabbed it while we were retrying
									log.Printf("DEBUG [RETRY %d/10]: Block was taken by another player. Aborting retry.", i)
									mu.Unlock()
									return
								}
								mu.Unlock()
							}
							
							// If we get here, 10 seconds passed and we failed. Clean up.
							log.Printf("[PERF ERR] Grab request %s timed out permanently after 10s.\n", reqID)
							network.PendingTimings.Delete(reqID)
							
							// Also clean up pending map so it doesn't leak memory
							mu.Lock()
							delete(state.Pending, reqID)
							mu.Unlock()
                    
               				}(b.ID, reqID, livePlayer.Id)
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

			// remove the player
			delete(state.Players, event.PlayerID)

			// figure out who takes over the blocks
			newOwner := getNextOwner(gameNet, event.NodeName)			

			// reassign orphaned blocks
			for _, b := range state.Blocks {
				if b.OwnerNode == event.NodeName {
					b.OwnerNode = newOwner

					// safety check: if the player who left was currently holding the block, drop it
					if b.HeldBy == event.PlayerID {
						b.HeldBy = ""
					}
				}
				log.Printf(
					"DEBUG [LEAVE]: Reassigning block %s owner %s -> %s",
					b.ID,
					event.NodeName,
					newOwner,
				)
			}

			for _, req := range state.Pending {
				if req.OwnerNode == event.NodeName {
					// transfer responsibility
					req.OwnerNode = newOwner
				}
				log.Printf(
					"DEBUG [LEAVE]: Reassigning pending req %s from %s -> %s",
					req.RequestID,
					event.NodeName,
					newOwner,
				)
			}

			mu.Unlock()

			if newOwner == gameNet.LocalName { 
				replayPendingRequests(state, gameNet)
			}
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

	case network.PENDING_REQ:
		reqID := parts[3]
		reqType := parts[4]
		blockID := parts[5]
		playerID := parts[6]
		dropX := MustAtoi(parts[7])
		dropY := MustAtoi(parts[8])
		ownerNode := parts[9]

		gameState.Pending[reqID] = &game.PendingRequest{
			RequestID: reqID,
			Type:      game.PendingRequestType(reqType),
			BlockID:   blockID,
			PlayerID:  playerID,
			DropX:     dropX,
			DropY:     dropY,
			OwnerNode: ownerNode,
			CreatedAt: time.Now(),
		}

	case network.GRAB_REQ:
		blockID := parts[3]
		playerID := parts[4]
		owner := parts[5]
		reqID := parts[6]

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

		var should_node_fail bool = false
		//for performance metrics - if nodefail, this node crashes. new owner should pick up
		if should_node_fail {
			os.Exit(1)
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
		gameNet.BroadcastGrabResult(reqID, blockID, playerID, success, b.OwnerNode)
		

	case network.GRAB_RES:
		log.Printf("DEBUG [GRAB_RES]: Received message! Raw parts: %v\n", parts)

		reqID := parts[3]
		blockID := parts[4]
		playerID := parts[5]
		success := parts[6] == "1"

		log.Printf("DEBUG [GRAB_RES]: Parsed - Block: %s, Player: %s, Success: %v\n", blockID, playerID, success)

		log.Printf(
			"[PERF LOAD] req=%s",
			reqID,
		)

		//end time, result received 
		if val, ok := network.PendingTimings.Load(reqID); ok {
			rt := val.(network.RequestTiming)

			latency := time.Since(rt.Start)

			nodefail := rt.NodeFail

			log.Printf(
				"[PERF] failed: %t. grab request %s completed in %v",
				nodefail,
				reqID,
				latency,
			)

			network.PendingTimings.Delete(reqID)
		} 
		
		if !success {
			log.Println("DEBUG [GRAB_RES]: Success was false, returning early.")
			delete(gameState.Pending, reqID)
			return
		}

		b, exists := gameState.Blocks[blockID]
		if !exists {
			log.Printf("DEBUG [GRAB_RES]: Block %s not found in gameState!\n", blockID)
			delete(gameState.Pending, reqID)
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

		delete(gameState.Pending, reqID)

	case network.DROP_REQ:
		blockID := parts[3]
		playerID := parts[4]
		dropX := MustAtoi(parts[5]) 
		dropY := MustAtoi(parts[6])
		owner := parts[7]

				req := &game.PendingRequest{
			RequestID: uuid.NewString(),
			Type:      game.PendingDrop,
			BlockID:   blockID,
			PlayerID:  playerID,
			DropX:     dropX,
			DropY:     dropY,
			OwnerNode: owner,
			CreatedAt: time.Now(),
		}

		gameNet.BroadcastPendingRequest(
			req.RequestID,
			req.BlockID,
			string(req.Type),
			req.PlayerID,
			req.OwnerNode,
			req.DropX,
			req.DropY,
		)

		gameState.Pending[req.RequestID] = req

		if owner != gameNet.LocalName {
			return
		}

		b, exists := gameState.Blocks[blockID]
		if !exists {
			return
		}

		//lock coordinate so can safely check if coords are occupied
		coordMu := gameState.GetCoordLock(dropX, dropY)
		coordMu.Lock()

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

		coordMu.Unlock()

		gameNet.BroadcastDropResult(req.RequestID, blockID, playerID, dropX, dropY, success)

	case network.DROP_RES:
		reqID := parts[3]
		blockID := parts[4]
		playerID := parts[5]
		dropX := MustAtoi(parts[6])
		dropY := MustAtoi(parts[7])
		success := parts[8] == "1"

		if !success {
			delete(gameState.Pending, reqID)
			return
		}

		b, bExists := gameState.Blocks[blockID]
		if !bExists {
			delete(gameState.Pending, reqID)
			return
		}

		// Unassign the block and update its coordinates
		b.HeldBy = ""
		b.Pos.X = dropX
		b.Pos.Y = dropY

		// Clear the block from the player, making them shrink back to 1x1
		if p, pExists := gameState.Players[playerID]; pExists {
			//double check player holding this block to prevent race conditions
			if p.HeldBlock != nil && p.HeldBlock.ID == blockID {
				p.HeldBlock = nil
			}
		}
		delete(gameState.Pending, reqID)

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

	case network.TEST_BLOCK_TIME:

	}




}

// todo: will this load a single node too much if lots leave? change to have a more spread out new owner distribution
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

	// sort nodes
	sort.Strings(activeNodes)

	//hash the id of the node leaving so new owner is spread out among all the nodes, no one node gets ownership of all orphaned blocks
	h := fnv.New32a()
	h.Write([]byte(leavingNode))

	idx := int(h.Sum32()) % len(activeNodes)

	return activeNodes[idx]
}

func replayPendingRequests(
	gameState *game.WorldState,
	gameNet *network.Network,
) {
	
	for _, req := range gameState.Pending {

		//debugging
		log.Printf(
			"DEBUG [REPLAY]: Replaying request %s type=%s owner=%s",
			req.RequestID,
			req.Type,
			req.OwnerNode,
		)

		if req.OwnerNode != gameNet.LocalName {
			continue
		}

		switch req.Type {

		case game.PendingGrab:
			replayGrab(gameState, gameNet, req)

		case game.PendingDrop:
			replayDrop(gameState, gameNet, req)
		}
	}
}

func replayGrab(gameState *game.WorldState, gameNet *network.Network, req *game.PendingRequest) {
	
	b, exists := gameState.Blocks[req.BlockID]
	if !exists {
		return
	}

	// already completed earlier
	if b.HeldBy == req.PlayerID {
		delete(gameState.Pending, req.RequestID)
		return
	}

	success := false

	if b.HeldBy == "" {
		b.HeldBy = req.PlayerID
		success = true

		if p, ok := gameState.Players[req.PlayerID]; ok {
			p.HeldBlock = b
		}
	}

	gameNet.BroadcastGrabResult(
		req.RequestID,
		req.BlockID,
		req.PlayerID,
		success,
		b.OwnerNode,
	)

	delete(gameState.Pending, req.RequestID)
}

func replayDrop(gameState *game.WorldState, gameNet *network.Network, req *game.PendingRequest) {
	b, exists := gameState.Blocks[req.BlockID]
	if !exists {
		return
	}
	
	// already completed earlier
	if b.HeldBy == "" && b.Pos.X == req.DropX && b.Pos.Y == req.DropY {
		delete(gameState.Pending, req.RequestID)
		return
	}
	
	success := false
	if b.HeldBy == req.PlayerID {
		b.HeldBy = ""
		b.Pos.X = req.DropX
		b.Pos.Y = req.DropY
		success = true

		if p, ok := gameState.Players[req.PlayerID]; ok {
			p.HeldBlock = nil
		}
	}

	gameNet.BroadcastDropResult(
		req.RequestID,
		req.BlockID,
		req.PlayerID,
		req.DropX,
		req.DropY,
		success,
	)

	delete(gameState.Pending, req.RequestID)
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

	// If join address provided join cluster
	if outgoing {
		joinAddr := os.Args[2]
		joinStart := time.Now()
		n, err := gameNet.List.Join([]string{joinAddr})
		if err != nil {
			log.Fatal(err)
		}
		joinLatency := time.Now().Sub(joinStart)
		fmt.Printf("Joined %d nodes (synced state over TCP in %v)\n", n, joinLatency)
	}

	fmt.Println("Node started:", machineName)

	return gameNet
}
