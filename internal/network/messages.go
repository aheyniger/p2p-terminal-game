package my_net

import (
	"fmt"
	"strings"
	"time"
	// "p2p-terminal-game/internal/game"
)

type MsgType int

const (
	JOIN MsgType = iota
	POS_UPDATE
	GRAB_REQ
	GRAB_RES
	DROP_REQ
	DROP_RES
	BLOCK_SPAWN
	STATE_SYNC
)

const Delim = "~|"

func buildMsg(delim string, fields ...any) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprint(f)
	}
	return strings.Join(parts, delim)
}

func (n *Network) BroadcastPosition(playerID string, x, y int) {
	// msg := fmt.Sprintf("%s|%d|%s|%d|%d",
	// 	n.List.LocalNode().Name,
	// 	time.Now().UnixNano(),
	// 	playerID,
	// 	x,
	// 	y,
	// )
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()
	msg := buildMsg(Delim,
		node,
		timestamp,
		POS_UPDATE,
		playerID,
		x,
		y,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}

func (n *Network) BroadcastJoin(playerID string, playerColor int32) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()
	msg := buildMsg(Delim,
		node,
		timestamp,
		JOIN,
		playerID,
		playerColor,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}

// broadcast grab requests by non owners
//now a direct send, not technically a broadcast
func (n *Network) BroadcastGrabRequest(blockID, playerID string, owner string) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()
	msg := buildMsg(Delim,
		node,
		timestamp,
		GRAB_REQ,
		blockID,
		playerID,
		owner,
	)

	n.SendDirect(owner, msg)
}

func (n *Network) BroadcastGrabResult(blockID, playerID string, success bool, owner string) {
	result := 0
	if success {
		result = 1
	}

	msg := buildMsg(Delim,
		n.LocalName,
		time.Now().UnixNano(),
		GRAB_RES,
		blockID,
		playerID,
		result,
	)

	n.Broadcast(msg)
}

func (n *Network) BroadcastDropRequest(blockID, playerID string, dropX, dropY int, owner string) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()
	msg := buildMsg(Delim,
		node,
		timestamp,
		DROP_REQ,
		blockID,
		playerID,
		dropX,
		dropY,
		owner,
	)

	// Send directly to the owner, same as GRAB_REQ
	n.SendDirect(owner, msg)
}

func (n *Network) BroadcastDropResult(blockID, playerID string, dropX, dropY int, success bool) {
	result := 0
	if success {
		result = 1
	}

	msg := buildMsg(Delim,
		n.LocalName,
		time.Now().UnixNano(),
		DROP_RES,
		blockID,
		playerID,
		dropX,
		dropY,
		result,
	)

	// Broadcast to everyone so they update the block's new position
	n.Broadcast(msg)
}

func (n *Network) BroadcastBlockSpawn(ID string, posX int, posY int, owner string) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()

	msg := buildMsg(Delim,
		node,
		timestamp,
		BLOCK_SPAWN,
		ID,
		posX,
		posY,
		owner,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}

func (n *Network) BroadcastStateSync(id string, posX int, posY int, owner string) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()

	msg := buildMsg(Delim,
		node,
		timestamp,
		STATE_SYNC,
		id,
		posX,
		posY,
		owner,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}
