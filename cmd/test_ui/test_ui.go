package main

import (
	"fmt"
	"p2p_game/internal/game"
	my_net "p2p_game/internal/network"
	"sync"

	// "time"

	"github.com/gdamore/tcell/v2"
)

func main() {

	ip := my_net.GetOutboundIP().String()

	net, err := my_net.CreateNetwork("node1", ip, 7946)
	if err != nil {
		panic(err)
	}

	var mu sync.Mutex

	state := &game.WorldState{
		Players: make(map[game.PlayerId]*game.Player),
	}

	wv := game.NewWorldView()
	defer wv.CloseWorldView()

	cx, cy := wv.GetViewCenter()

	localPlayer := &game.Player{
		Id:    "local",
		Color: 0x0000FF,
		Pos:   game.Vec2{X: cx, Y: cy},
	}

	state.Players[localPlayer.Id] = localPlayer

	// NETWORK HOOK (incoming msgs)
	net.OnPositionUpdate = func(id string, x, y int) {
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
		net.BroadcastPosition(p.Id, p.Pos.X, p.Pos.Y)

		wv.Ui.SetHeaderField("Last input", lastKey)
		wv.Ui.SetHeaderField("Location",
			fmt.Sprintf("%v,%v", p.Pos.X, p.Pos.Y),
		)

		w, h := wv.GetViewSize()
		wv.Ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", w, h))

		return false
	}

	go func() {
		wv.RenderLoop(state, keyInputHandler)
	}()

	select {} // block forever
}
