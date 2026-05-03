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

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}

func (n *Network) BroadcastGrabResult(blockID, playerID string, success bool) {
	result := 0
	if success {
		result = 1
	}

	msg := fmt.Sprintf("%s|%d|%d|%s|%s|%d",
		n.LocalName,
		time.Now().UnixNano(),
		GRAB_RES,
		blockID,
		playerID,
		result,
	)

	n.Broadcast(msg)
}

func (n *Network) BroadcastBlockSpawn(ID string, posX int, posY int) {
	node := n.List.LocalNode().Name
	timestamp := time.Now().UnixNano()

	msg := buildMsg(Delim,
		node,
		timestamp,
		BLOCK_SPAWN,
		ID,
		posX,
		posY,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}