package game

import (
	"fmt"
	tui "p2p_game/internal/ui"
	"time"

	"github.com/gdamore/tcell/v2"
)

type View struct {
	Ui tui.Ui
}

func NewWorldView() View {
	view := View{Ui: tui.GetScreen()}
	ui := view.Ui

	ui.Screen.SetStyle(tui.DefStyle)
	ui.Screen.Fill(' ', tui.DefStyle)

	return view
}

func (view *View) CloseWorldView() {
	view.Ui.Quit()
}

func (view *View) RenderLoop(state *WorldState, keyInputHandler func(e *tcell.EventKey) bool) {
	ui := view.Ui
	//TODO: should this be in just the ui package?
	events := make(chan tcell.Event)
	go func() {
		for {
			events <- ui.Screen.PollEvent()
		}
	}()
	// Run UI at 30fps
	ticker := time.NewTicker(33 * time.Millisecond)
	for {
		select {
		case ev := <-events:
			switch e := ev.(type) {
			case *tcell.EventKey:
				finish := keyInputHandler(e)
				if finish {
					return
				}
			case *tcell.EventResize:
				width, height := view.GetViewSize()
				ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", width, height))
				ui.Screen.Sync()
			}
		case <-ticker.C:
			//ui.Screen.Clear()
			view.DrawWorld(*state)
			ui.Screen.Show()
		}
	}
}

// TODO: how should redrawing be done? Should it just reset the whole screen and keep redrawing everything, or only changes?
// TODO: if only changes, should DrawPlayer also remove the player's old position?
func (view *View) DrawWorld(state WorldState) {
	view.Ui.Screen.Fill(' ', tui.DefStyle)

	for id, player := range state.Players {
		_ = id
		view.DrawPlayer(*player)
	}

	view.Ui.DrawHeader()
}

func (view *View) DrawPlayer(player Player) {
	view.Ui.DrawTile(player.Pos.X, player.Pos.Y)
}

func (view View) GetViewCenter() (int, int) {
	width, height := view.Ui.Screen.Size()
	return width / 2, height / 2
}

func (view View) GetViewSize() (int, int) {
	width, height := view.Ui.Screen.Size()
	return width / 2, height / 2
}
