package my_net

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	. "p2p_game/internal/misc"

	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
)

type LeaveEvent struct {
	NodeName string
	PlayerID string
}

type Network struct {
	List  *memberlist.Memberlist
	Queue *memberlist.TransmitLimitedQueue

	NodePlayers      map[string]string
	seen             map[string]bool
	OnMsg            func([]byte)
	OnPositionUpdate func(id string, x, y int)
	LocalName        string
	LeaveEventCh     chan LeaveEvent

	//for tcp state sync (slower but more reliable than udp. still use udp for update messages)
	GetLocalState func() []byte
	MergeState    func([]byte)

	// for performance benchmarking
	PendingAcks sync.Map
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

func (d *delegate) NodeMeta(limit int) []byte { return nil }
func (d *delegate) LocalState(join bool) []byte {
	// world state to send over TCP
	if d.net.GetLocalState != nil {
		return d.net.GetLocalState()
	}
	return []byte{}
}
func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	//  TCP state from the cluster, to merge
	if d.net.MergeState != nil {
		d.net.MergeState(buf)
	}
}

func (d *delegate) NotifyMsg(msg []byte) {
	if d.net.OnMsg != nil {
		d.net.OnMsg(msg)
	}
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.net.Queue.GetBroadcasts(overhead, limit)
}

type EventDelegate struct {
	mu  sync.Mutex
	net *Network
}

func (e *EventDelegate) NotifyJoin(node *memberlist.Node) {
	// e.mu.Lock()
	// defer e.mu.Unlock()

	log.Printf("Node joined: %s (%s)", node.Name, node.Addr)
}

func (e *EventDelegate) NotifyLeave(node *memberlist.Node) {
	e.mu.Lock()
	playerId := e.net.NodePlayers[node.Name]
	delete(e.net.NodePlayers, node.Name)
	e.mu.Unlock()

	go func() {
		e.net.LeaveEventCh <- LeaveEvent{
			NodeName: node.Name,
			PlayerID: playerId,
		}
	}()

	log.Printf("Node left: %s (%s)", node.Name, node.Addr)
}

// SendDirect sends a guaranteed, instant TCP message to a single specific node
func (n *Network) SendDirect(targetNodeName string, msg string) {
	for _, node := range n.List.Members() {
		if node.Name == targetNodeName {
			// Bypasses the gossip queue entirely
			n.List.SendReliable(node, []byte(msg))
			return
		}
	}
}

func (e *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	log.Printf("Node updated: %s", node.Name)
}

func CreateNetwork(name string, bindIP string, port int, logCh chan string) (*Network, error) {
	config := memberlist.DefaultLANConfig()

	config.Name = name
	config.BindAddr = "0.0.0.0"
	config.BindPort = port
	config.AdvertiseAddr = bindIP
	config.AdvertisePort = port

	n := &Network{
		seen:        make(map[string]bool),
		LocalName:   name,
		NodePlayers: make(map[string]string),
	}

	writer := NewChanWriter(logCh)
	logger := log.New(writer, "[memberlist] ", 0)

	config.Logger = logger

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

type NodeTime struct {
	AckTime  time.Time
	NodeName string
}

func (n *Network) TestBroadcastAndMeasure() time.Duration {
	var acksReceived map[string]struct{}
	acksReceived = make(map[string]struct{})

	msgId := uuid.NewString()
	expected := n.List.NumMembers() - 1

	acks := make(chan NodeTime, expected)
	n.PendingAcks.Store(msgId, acks)

	msg := buildMsg(Delim,
		n.LocalName,
		time.Now().UnixNano(),
		TEST_GOSSIP,
		msgId,
	)
	n.Broadcast(msg)

	broadcastTime := time.Now()

	received := 0
	var lastAck time.Time
	var lastAckNode string
	timeout := time.After(10 * time.Second)
	for received < expected {
		select {
		case nt := <-acks:
			if _, ok := acksReceived[nt.NodeName]; !ok {
				received++
				acksReceived[nt.NodeName] = struct{}{}
				if nt.AckTime.After(lastAck) {
					lastAck = nt.AckTime
					lastAckNode = nt.NodeName
				}
			}
		case <-timeout:
			log.Printf("only %d/%d acks received\n", received, expected)
			goto done
		}
	}

done:

	convergenceTime := lastAck.Sub(broadcastTime)

	n.PendingAcks.Delete(msgId)
	log.Printf("Last ack came from %s and took %v\n", lastAckNode, convergenceTime)

	var rtt time.Duration
	var err error
	for _, node := range n.List.Members() {
		if node.Name == lastAckNode {
			addr := &net.UDPAddr{
				IP:   node.Addr,
				Port: int(node.Port),
			}
			rtt, err = n.List.Ping(node.Name, addr)
			if err == nil {
				log.Printf("RTT to %s: %v\n", node.Name, rtt)
			}
			break
		}
	}

	if rtt != 0 {
		convergenceTime = convergenceTime - rtt/2
	}

	return convergenceTime
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
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {}
