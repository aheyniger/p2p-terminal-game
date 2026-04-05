package main

import (
	"fmt"
	tui "p2p_game/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func main() {
	// samplerUi()
	ui := tui.GetScreen()
	ui.Screen.SetStyle(tui.DefStyle)
	// ui.Screen.Clear()
	ui.Screen.Fill('', )

	defer ui.Quit()

	s := ui.Screen

	width, height := s.Size()

	x, y := width/2, height/2
	px, py := x, y

	ui.DrawTile(x, y)

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
		ui.DrawTile(x, y)
		px, py = x, y

		ui.SetHeaderField("Location", fmt.Sprintf("%v,%v", x, y))
		ui.SetHeaderField("Dimensions", fmt.Sprintf("%vWx%vH", width, height))
		ui.DrawHeader()

		ui.Show()
	}
}

func samplerUi() {
	ui := tui.GetScreen()

	ui.Screen.SetStyle(tui.DefStyle)
	ui.Screen.EnableMouse()
	ui.Screen.EnablePaste()
	ui.Screen.Clear()

	defer ui.Quit()

	s := ui.Screen

	ox, oy := -1, -1
	for {
		ui.Show()

		ev := ui.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			ui.Screen.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				s.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				s.Clear()
			}
			// else if ev.Rune() == 'B' {}
		case *tcell.EventMouse:
			x, y := ev.Position()

			switch ev.Buttons() {
			case tcell.Button1, tcell.Button2:
				if ox < 0 {
					ox, oy = x, y // record location when click started
				}

			case tcell.ButtonNone:
				if ox >= 0 {
					label := fmt.Sprintf("%d,%d to %d,%d", ox, oy, x, y)
					tui.DrawBox(s, ox, oy, x, y, tui.BoxStyle, label)
					ox, oy = -1, -1
				}
			}
		}
	}
}
