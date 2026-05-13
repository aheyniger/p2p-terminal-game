package game

import "sync"

const NumCoordLocks = 64

var coordLocks [NumCoordLocks]sync.Mutex

type PendingRequestType string

const (
	PendingGrab PendingRequestType = "grab"
	PendingDrop PendingRequestType = "drop"
)

func (w *WorldState) MovePlayer(id PlayerId, dx, dy int) {
	p, ok := w.Players[id]
	if !ok {
		return
	}

	p.Pos.X += dx
	p.Pos.Y += dy
	w.Version++
}

// func (w *WorldState) TryGrabBlock(playerID PlayerId) {
// 	p := w.Players[playerID]
// 	if p == nil || p.HeldBlock != nil {
// 		return
// 	}

// 	for _, b := range w.Blocks {
// 		if b.HeldBy == "" && b.Pos == p.Pos {
// 			b.HeldBy = playerID
// 			p.HeldBlock = b
// 			return
// 		}
// 	}
// }

func (w *WorldState) FindBlockByID(playerID PlayerId) *Block {
	for _, b := range w.Blocks {
		if b.ID == playerID {
			return b
		}
	}
	return nil
}

func (w *WorldState) GetCoordLock(x, y int) *sync.Mutex {
	// A standard spatial hashing formula using large primes
	hash := (x * 73856093) ^ (y * 19349663)
	
	// Ensure the hash is positive before modulo
	if hash < 0 {
		hash = -hash
	}
	
	return &coordLocks[hash%NumCoordLocks]
}

