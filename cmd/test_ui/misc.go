package main

import (
	"fmt"
	tui "p2p_game/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func BasicTestUi() {
	ui := tui.GetScreen()
	ui.Screen.SetStyle(tui.DefStyle)
	// ui.Screen.Clear()
	ui.Screen.Fill(' ', tui.DefStyle)

	defer ui.Quit()

	s := ui.Screen

	width, height := s.Size()

	x, y := width/2, height/2
	px, py := x, y

	ui.DrawTile(x, y, 0xFF0000)

	ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", x, y))
	ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", width, height))
	ui.SetHeaderField("Last input", "")
	ui.DrawHeader()

	ui.Show()
	for {

		ev := ui.Screen.PollEvent()
		switch e := ev.(type) {
		case *tcell.EventResize:
			ui.Screen.Sync()
			width, height = s.Size()
		case *tcell.EventKey:
			var lastKey string

			switch e.Key() {
			case tcell.KeyEscape:
				return
			case tcell.KeyUp:
				lastKey = "UpArrow"
				y--
			case tcell.KeyDown:
				lastKey = "DownArrow"
				y++
			case tcell.KeyLeft:
				lastKey = "LeftArrow"
				x--
			case tcell.KeyRight:
				lastKey = "RightArrow"
				x++
			case tcell.KeyRune:
				lastKey = (string)(e.Rune())
				switch e.Rune() {
				case 'r', 'R':
					x, y = width/2, height/2
				}
			}
			ui.SetHeaderField("Last input", lastKey)
		}

		ui.EraseTile(px, py)
		ui.DrawTile(x, y, 0xFF0000)
		px, py = x, y

		ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", x, y))
		ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", width, height))
		ui.DrawHeader()

		ui.Show()
	}
}
