package tui

import (
	"github.com/gdamore/tcell/v2"
)

func (ui *Ui) DrawTile(x, y int) {
	s := ui.Screen

	// s.Put(x, y, " ", BoxStyle)
	s.SetContent(x, y, '█', nil, NoBgStyle)
}

func (ui *Ui) EraseTile(x, y int) {
	s := ui.Screen

	// s.Put(x, y, " ", DefStyle)
	s.SetContent(x, y, ' ', nil, DefStyle)
}

func (ui *Ui) DrawTopTile(x, y int) {
	s := ui.Screen

	// s.Put(x, y, "\u0305", BoxStyle)
	s.SetContent(x, y, '▀', nil, NoBgStyle)
}

func (ui *Ui) DrawBottomTile(x, y int) {
	s := ui.Screen

	// s.Put(x, y, "\u0305", BoxStyle)
	s.SetContent(x, y, '▄', nil, NoBgStyle)
}

func (ui *Ui) DrawText(x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	var width int
	for text != "" {
		text, width = ui.Screen.Put(col, row, text, style)
		col += width
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
		if width == 0 {
			break
		}
	}
}

func (ui *Ui) DrawHeader() {
	//TODO: make the ordering of the header fields fixed, so they don't randomly change order (which happens due to ui.headerFields being a map, which is unordered)
	s := ui.Screen

	// headerHeight := 1

	width, _ := s.Size()

	for col := 0; col <= width; col++ {
		s.Put(col, 0, " ", HeaderStyle)
	}

	numFields := len(ui.headerFields)
	if numFields == 0 {
		return
	}

	labelInterval := width / numFields

	i := 0
	for _, fieldName := range ui.headerFields {
		ui.DrawText(i, 0, width, 0, HeaderStyle, fieldName+": ")
		ui.DrawText(i+len(fieldName)+2, 0, width, 0, HeaderStyle, ui.headerFieldValues[fieldName])
		i += labelInterval
	}
}
