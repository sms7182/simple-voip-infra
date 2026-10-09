package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	voipinfra "voip.infra"
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
	ctx := context.Background()

	registrar := voipinfra.NewRegistrar()

	go registrar.Cleanup(ctx)
	fmt.Println("udp server listening :5060")
	buffer := make([]byte, 2048)

	for {
		n, clientAddress, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("read error:%v", err)
			continue
		}
		rawRequest := string(buffer[:n])
		if strings.HasPrefix(rawRequest, "REGISTER") {
			handleRegister(rawRequest, *registrar, conn, clientAddress)
			continue
		}
		if strings.HasPrefix(rawRequest, "INVITE") {
			handleInvite(rawRequest, *registrar, conn, clientAddress)
			continue
		}
		_, err = conn.WriteToUDP(buffer[:n], clientAddress)

	}
}

func handleInvite(rawRequest string, registrar voipinfra.Registrar, conn *net.UDPConn, clientAddress *net.UDPAddr) {

}
func handleRegister(rawRequest string, registrar voipinfra.Registrar, conn *net.UDPConn, clientAddress *net.UDPAddr) {
	callID := getHeader(rawRequest, "Call-ID:")
	cseq := getHeader(rawRequest, "CSeq:")
	from := getHeader(rawRequest, "From:")
	to := getHeader(rawRequest, "To:")
	via := getHeader(rawRequest, "Via:")
	contact := getHeader(rawRequest, "Contact:")

	username := extractUsername(to)
	regis := voipinfra.Registration{
		Username:  username,
		Contact:   contact,
		IP:        clientAddress.IP,
		Port:      clientAddress.Port,
		From:      from,
		To:        to,
		Via:       via,
		ExpiresAt: time.Now().Add(time.Duration(3600) * time.Second),
	}

	registrar.Register(regis)

	response := fmt.Sprintf(
		"SIP/2.0 200 OK\r\n"+
			"Via: %s\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"Call-ID: %s\r\n"+
			"CSeq: %s\r\n"+
			"Contact: %s\r\n"+
			"Content-Length: 0\r\n\r\n",
		via, from, to, callID, cseq, contact,
	)
	_, err := conn.WriteToUDP([]byte(response), clientAddress)
	if err != nil {
		log.Printf("write error:%v", err)
	}
}

func getHeader(sipMsg, header string) string {
	lines := strings.Split(sipMsg, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), header) {
			return strings.TrimSpace(strings.TrimPrefix(line, header))
		}
	}
	return ""
}
func extractUsername(toHeader string) string {
	if strings.Contains(toHeader, "sip:") {
		parts := strings.Split(toHeader, "sip:")
		if len(parts) > 1 {
			userDomain := strings.Split(parts[1], "@")[0]
			return strings.Trim(userDomain, "<> ")
		}
	}
	return toHeader
}
