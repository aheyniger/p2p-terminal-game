package game

func (w *WorldState) MovePlayer(id PlayerId, dx, dy int) {
	p, ok := w.Players[id]
	if !ok {
		return
	}

	p.Pos.X += dx
	p.Pos.Y += dy
	w.Version++
}

