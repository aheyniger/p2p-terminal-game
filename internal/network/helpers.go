package my_net

import (
	"fmt"
	"log"
	"net"
)

func GetOutboundIP() net.IP {
	// The address doesn't need to exist; UDP is connectionless
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

// Helper func to convert string → int
func MustAtoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
