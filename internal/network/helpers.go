package my_net

import (
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
