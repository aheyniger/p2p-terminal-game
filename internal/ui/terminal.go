package tui

import (
	"log"

	"github.com/gdamore/tcell/v2"
)

type Ui struct {
	Screen tcell.Screen
}

func NewScreenTest() {
	screen, err := tcell.NewScreen()

	if err != nil {
		log.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}
	defer screen.Fini()

	screen.SetContent(0, 0, 'A', nil, tcell.StyleDefault)
	screen.Show()

	for {
		ev := screen.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}
}

func GetScreen() Ui {
	screen, err := tcell.NewScreen()

	if err != nil {
		log.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}

	return Ui{Screen: screen}
}

func (ui *Ui) Quit() {
	s := ui.Screen
	maybePanic := recover()

	s.Fini()
	if maybePanic != nil {
		panic(maybePanic)
	}
}

func (ui *Ui) Show() {
	ui.Screen.Show()
}
