package tui

func (ui *Ui) DrawTile(x, y int) {
	s := ui.Screen

	s.Put(x, y, " ", BoxStyle)
}

func (ui *Ui) EraseTile(x, y int) {
	s := ui.Screen

	s.Put(x, y, " ", DefStyle)
}
