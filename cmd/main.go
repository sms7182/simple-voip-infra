package main

import (
	"fmt"
	"log"
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
	buffer := make([]byte, 1024)
	for {
		n, clientAddress, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("read error:%v", err)
			continue
		}
		fmt.Printf("Got message from %s:%s\n", clientAddress, string(buffer[:n]))
		_, err = conn.WriteToUDP(buffer[:n], clientAddress)
		if err != nil {
			log.Printf("write error:%v", err)
		}
	}
}
