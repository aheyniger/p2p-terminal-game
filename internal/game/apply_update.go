package game

func (w *WorldState) ApplyRemoteUpdate(id string, x, y int) {
	p, ok := w.Players[id]
	if !ok {
		p = &Player{Id: id, Pos: Vec2{X: x, Y: y}}
		w.Players[id] = p
		return
	}

	p.Pos.X = x
	p.Pos.Y = y
}