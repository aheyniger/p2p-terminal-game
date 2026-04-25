package my_net

import (
	"fmt"
	"strings"
	"time"
)

type MsgType int

const (
	JOIN MsgType = iota
	POS_UPDATE
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
