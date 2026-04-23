package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	network "p2p_game/internal/network"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
)

var seen_msg = make(map[string]bool) //map of seen messages so we can deduplicate

func main() {

	queue := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return 3 // or len(list.Members())
		},
		RetransmitMult: 3,
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run main.go <port> [join_ip:port]")
		return
	}

	port := mustAtoi(os.Args[1])

	nodeIP := os.Getenv("NODE_IP")
	if nodeIP == "" {
		nodeIP = fmt.Sprintf("%v", network.GetOutboundIP())
	}
	fmt.Println("Outbound IP:", nodeIP)

	// Create default config
	config := memberlist.DefaultLANConfig()

	config.BindPort = port
	config.BindAddr = "0.0.0.0"

	config.AdvertiseAddr = nodeIP
	// config.AdvertiseAddr = nodeIP
	config.AdvertisePort = port

	config.Delegate = &Delegate{queue: queue}

	// void load_config(){
	//     nodes[0] = {"128.180.120.95", 4041}; //neptune: 128.180.120.95
	//     nodes[1] = {"128.180.120.73", 4041}; //eris: 128.180.120.73
	//     nodes[2] = {"128.180.120.86", 4041}; //puck: 128.180.120.86
	//     nodes[3] = {"128.180.120.76", 4041}; //iapetus: 128.180.120.76
	// }

	config.Name = fmt.Sprintf("%s-%d", nodeIP, port)

	// Create memberlist
	list, err := memberlist.Create(config)
	if err != nil {
		log.Fatal(err)
	}

	// If join address provided → join cluster
	if len(os.Args) > 2 {
		joinAddr := os.Args[2]
		n, err := list.Join([]string{joinAddr})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Joined %d nodes\n", n)
	}

	fmt.Println("Node started:", config.Name)

	// Keep running + print members periodically
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter log: ")
		input, _ := reader.ReadString('\n')

		input = strings.TrimSpace(input)

		if input != "" {
			broadcastLog(queue, config.Name, input)
		}
	}
}

type LogBroadcast struct {
	msg []byte
	// notify chan struct{}
}

func (l *LogBroadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (l *LogBroadcast) Message() []byte {
	return l.msg
}

func (l *LogBroadcast) Finished() {

}

type Delegate struct {
	queue *memberlist.TransmitLimitedQueue
}

func (d *Delegate) NodeMeta(limit int) []byte              { return nil }
func (d *Delegate) LocalState(join bool) []byte            { return nil }
func (d *Delegate) MergeRemoteState(buf []byte, join bool) {}

// handler when receiving a message
func (d *Delegate) NotifyMsg(msg []byte) {
	// log.Printf("DEBUG NotifyMsg called: %s\n", string(msg))
	logLine := string(msg)

	parts := strings.SplitN(logLine, "|", 3)
	if len(parts) != 3 {
		log.Println("Malformed message:", logLine)
		return
	}

	node := parts[0]
	id := parts[1]
	message := parts[2]

	// Deduplicate using ID
	if seen_msg[id] {
		return
	}
	seen_msg[id] = true

	// Clean display
	formatted := fmt.Sprintf("[%s] %s", node, message)

	fmt.Println(formatted)

	// Append to file
	f, err := os.OpenFile("shared.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("File error:", err)
		return
	}
	defer f.Close()

	f.WriteString(logLine + "\n")
}

func broadcastLog(queue *memberlist.TransmitLimitedQueue, nodeName string, message string) {
	full := fmt.Sprintf("%s|%d|%s",
		nodeName,
		time.Now().UnixNano(),
		message,
	) //added a unique ID to each message so we can filter duplicates for the log

	queue.QueueBroadcast(&LogBroadcast{
		msg: []byte(full),
	})
}

// runs this before sending a broadcast
func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	msgs := d.queue.GetBroadcasts(overhead, limit)
	// log.Printf("DEBUG GetBroadcasts called, returning %d messages\n", len(msgs))
	return msgs
}

// Helper func to convert string → int
func mustAtoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// Print cluster members
func printMembers(list *memberlist.Memberlist) {
	fmt.Println("Current members:")
	for _, member := range list.Members() {
		fmt.Printf(" - %s (%s)\n",
			member.Name,
			member.Address())
	}
	fmt.Println(strings.Repeat("-", 20))
}
