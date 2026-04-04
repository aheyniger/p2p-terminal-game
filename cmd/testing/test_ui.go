package main

import (
	"fmt"
	tui "p2p_game/internal/ui"

	"github.com/gdamore/tcell/v2"
)

func main() {
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
