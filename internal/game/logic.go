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

func (w *WorldState) TryGrabBlock(playerID PlayerId) {
	p := w.Players[playerID]
	if p == nil || p.HeldBlock != nil {
		return
	}

	for _, b := range w.Blocks {
		if b.HeldBy == "" && b.Pos == p.Pos {
			b.HeldBy = playerID
			p.HeldBlock = b
			return
		}
	}
}

func (w *WorldState) FindBlockByID(playerID PlayerId) *Block {
	for _, b := range w.Blocks {
		if b.ID == playerID {
			return b
		}
	}
	return nil
}
