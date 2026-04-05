package main

import (
	"fmt"
	"p2p_game/internal/game"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

func main() {
	// SamplerUi()
	// BasicTestUi()

	state := game.WorldState{}
	state.Players = make(map[game.PlayerId]*game.Player)

	wv := game.NewWorldView()
	defer wv.CloseWorldView()
	// time.Sleep(1000 * time.Millisecond)

	cx, cy := wv.GetViewCenter()
	localPlayer := game.Player{Id: "local", Color: "blue", Pos: game.Vec2{X: cx, Y: cy}}
	lp := &localPlayer
	state.Players[localPlayer.Id] = &localPlayer
	wv.Ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", lp.Pos.X, lp.Pos.Y))

	// TODO: can we find a way to not need to use tcell at all in this class?
	keyInputHandler := func(e *tcell.EventKey) bool {
		lastKey := ""

		//TODO: maybe what this should instead be is a struct of function handlers that you register
		switch e.Key() {
		case tcell.KeyEscape:
			return true
		case tcell.KeyUp:
			lastKey = "UpArrow"
			lp.Pos.Y--
		case tcell.KeyDown:
			lastKey = "DownArrow"
			lp.Pos.Y++
		case tcell.KeyLeft:
			lastKey = "LeftArrow"
			lp.Pos.X--
		case tcell.KeyRight:
			lastKey = "RightArrow"
			lp.Pos.X++
		case tcell.KeyRune:
			lastKey = (string)(e.Rune())
			switch e.Rune() {
			case 'r', 'R':
				width, height := wv.GetViewSize()
				lp.Pos.X, lp.Pos.Y = width/2, height/2
			}
		}
		wv.Ui.SetHeaderField("Last input", lastKey)
		wv.Ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", lp.Pos.X, lp.Pos.Y))
		width, height := wv.GetViewSize()
		wv.Ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", width, height))
		return false
	}
	// time.Sleep(1000 * time.Millisecond)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		wv.RenderLoop(&state, keyInputHandler)
	}()

	wg.Wait()
	time.Sleep(1 * time.Millisecond)

}
