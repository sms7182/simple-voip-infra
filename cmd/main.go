package main

import (
	"fmt"
	"net"
)

func main() {

	address, err := net.ResolveUDPAddr("udp", "0.0.0.0:5060")
	if err != nil {
		fmt.Println("Error resolving  address ", err)
		return
	}

	conn, err := net.ListenUDP("udp", address)
	if err != nil {
		fmt.Println("error listening ", err)
		return
	}
	defer conn.Close()
	fmt.Println("udp server listening :5060")
}
