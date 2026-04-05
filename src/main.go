package main
import "bufio"

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/memberlist"
)

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
		nodeIP = "127.0.0.1"
	}

	// Create default config
	config := memberlist.DefaultLANConfig()

	config.BindPort = port
	config.BindAddr = "0.0.0.0"

	// config.AdvertiseAddr = os.Getenv("NODE_IP")
	config.AdvertiseAddr = nodeIP
	config.AdvertisePort = port


	config.Delegate = &Delegate{queue: queue}

/*
void load_config(){
    nodes[0] = {"128.180.120.95", 6005}; //neptune: 128.180.120.95
    nodes[1] = {"128.180.120.73", 6005}; //eris: 128.180.120.73
    nodes[2] = {"128.180.120.86", 6005}; //puck: 128.180.120.86
    nodes[3] = {"128.180.120.76", 6005}; //iapetus: 128.180.120.76
}	*/

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
	for {
		var input string
		fmt.Print("Enter log: ")
		
		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Print("Enter log: ")
			input, _ := reader.ReadString('\n')

			input = strings.TrimSpace(input)

			if input != "" {
				broadcastLog(queue, input)
			}
		}

		broadcastLog(queue, input)
	}
}

type LogBroadcast struct {
	msg    []byte
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

type Delegate struct{
	queue *memberlist.TransmitLimitedQueue
}

func (d *Delegate) NodeMeta(limit int) []byte { return nil }
func (d *Delegate) LocalState(join bool) []byte { return nil }
func (d *Delegate) MergeRemoteState(buf []byte, join bool) {}

func (d *Delegate) NotifyMsg(msg []byte) {
	logLine := string(msg)

	fmt.Println("Received log:", logLine)

	// Append to file
	f, err := os.OpenFile("shared.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("File error:", err)
		return
	}
	defer f.Close()

	f.WriteString(logLine + "\n")
}

func broadcastLog(queue *memberlist.TransmitLimitedQueue, message string) {
	b := &LogBroadcast{
		msg: []byte(message),
	}
	queue.QueueBroadcast(b)
}

func (d *Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.queue.GetBroadcasts(overhead, limit)
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