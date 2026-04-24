package my_net

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)

type Network struct {
	List  *memberlist.Memberlist
	Queue *memberlist.TransmitLimitedQueue

	seen             map[string]bool
	OnMsg            func([]byte)
	OnPositionUpdate func(id string, x, y int)
	LocalName        string
}

type delegate struct {
	net *Network
}

type Message struct {
	ID   string
	Type string
	X    int
	Y    int
}

func (d *delegate) NodeMeta(limit int) []byte              { return nil }
func (d *delegate) LocalState(join bool) []byte            { return nil }
func (d *delegate) MergeRemoteState(buf []byte, join bool) {}

func (d *delegate) NotifyMsg(msg []byte) {
	if d.net.OnMsg != nil {
		d.net.OnMsg(msg)
	}
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.net.Queue.GetBroadcasts(overhead, limit)
}

type EventDelegate struct {
	mu sync.Mutex
	// members map[string]*memberlist.Node
	net *Network
}

func (e *EventDelegate) NotifyJoin(node *memberlist.Node) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// e.members[node.Name] = node
	log.Printf("Node joined: %s (%s)", node.Name, node.Addr)
}

func (e *EventDelegate) NotifyLeave(node *memberlist.Node) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// delete(e.members, node.Name)
	log.Printf("Node left: %s (%s)", node.Name, node.Addr)
}

func (e *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	log.Printf("Node updated: %s", node.Name)
}

func CreateNetwork(name string, bindIP string, port int) (*Network, error) {
	config := memberlist.DefaultLANConfig()

	config.Name = name
	config.BindAddr = "0.0.0.0"
	config.BindPort = port
	config.AdvertiseAddr = bindIP
	config.AdvertisePort = port

	n := &Network{
		seen:      make(map[string]bool),
		LocalName: name,
	}

	queue := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			if n.List == nil {
				return 1
			}
			return len(n.List.Members())
		},
		RetransmitMult: 3,
	}

	n.Queue = queue
	config.Delegate = &delegate{net: n}
	config.Events = &EventDelegate{net: n}

	list, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	n.List = list
	return n, nil
}

// func broadcastLog(queue *memberlist.TransmitLimitedQueue, nodeName string, message string) {
// 	full := fmt.Sprintf("%s|%d|%s",
// 		nodeName,
// 		time.Now().UnixNano(),
// 		message,
// 	) //added a unique ID to each message so we can filter duplicates for the log

// 	queue.QueueBroadcast(&LogBroadcast{
// 		msg: []byte(full),
// 	})
// }

func (n *Network) BroadcastPosition(playerID string, x, y int) {
	msg := fmt.Sprintf("%s|%d|%s|%d|%d",
		n.List.LocalNode().Name,
		time.Now().UnixNano(),
		playerID,
		x,
		y,
	)

	n.Queue.QueueBroadcast(&broadcast{
		msg: []byte(msg),
	})
}

func buildMoveMessage(playerID string, x, y int) string {
	return fmt.Sprintf("%s|%d|%d", playerID, x, y)
}

// func (n *Network) Join(addresses []string) error {
// 	_, err := n.List.Join(addresses)
// 	return err
// }

func (n *Network) Broadcast(msg string) {
	b := &broadcast{
		msg: []byte(msg),
	}
	n.Queue.QueueBroadcast(b)
}

type broadcast struct {
	msg []byte
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return true
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {}
