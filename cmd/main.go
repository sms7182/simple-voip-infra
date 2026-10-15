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

var activeCalls = make(map[string]voipinfra.CallSession)

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
			handleRegister(rawRequest, registrar, conn, clientAddress)
			continue
		}
		if strings.HasPrefix(rawRequest, "INVITE") {
			handleInvite(rawRequest, registrar, conn, clientAddress)
			continue
		}
		if strings.HasPrefix(rawRequest, "SIP/2.0 180") ||
			strings.HasPrefix(rawRequest, "SIP/2.0 200") ||
			strings.HasPrefix(rawRequest, "ACK") {
			handleRelay(rawRequest, registrar, clientAddress, conn)
			continue
		}
		_, err = conn.WriteToUDP(buffer[:n], clientAddress)

	}
}

func handleRelay(rawRequest string, register *voipinfra.Registrar, clientAddress *net.UDPAddr, conn *net.UDPConn) {
	callID := getHeader(rawRequest, "Call-ID:")
	if callID == "" {
		return
	}
	activeSession, ok := activeCalls[callID]
	if !ok {
		log.Printf("relay error: no active session found for Call-ID %s", callID)
		return
	}
	clientStr := clientAddress.String()
	if activeSession.CalleeAddr.String() == clientStr {
		_, err := conn.WriteToUDP([]byte(rawRequest), activeSession.CallerAddr)
		if err != nil {
			log.Printf("Error relaying response to caller: %v", err)
		} else {
			log.Printf("Relayed response from Callee to Caller at %s", activeSession.CallerAddr.String())
		}
	} else if activeSession.CallerAddr.String() == clientStr {
		_, err := conn.WriteToUDP([]byte(rawRequest), activeSession.CalleeAddr)
		if err != nil {
			log.Printf("Error relaying request to callee: %v", err)
		} else {
			log.Printf("Relayed ACK/request from Caller to Callee at %s", activeSession.CalleeAddr.String())
		}
	} else {
		log.Printf("Relay warning: Packet from unknown source address: %s", clientStr)
	}
}

func handleInvite(rawRequest string, registrar *voipinfra.Registrar, conn *net.UDPConn, clientAddress *net.UDPAddr) {
	callID := getHeader(rawRequest, "Call-ID:")
	cseq := getHeader(rawRequest, "CSeq:")
	from := getHeader(rawRequest, "From:")
	to := getHeader(rawRequest, "To:")
	via := getHeader(rawRequest, "Via:")
	_ = getHeader(rawRequest, "Contact:")

	username := extractUsername(to)
	target, ok := registrar.Get(username)
	if !ok {
		notFoundResp := fmt.Sprintf("SIP/2.0 404 Not Found\r\n"+
			"Via: %s\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"Call-ID: %s\r\n"+
			"CSeq: %s\r\n"+
			"Content-Length: 0\r\n\r\n",
			via, from, to, callID, cseq,
		)
		conn.WriteToUDP([]byte(notFoundResp), clientAddress)
		log.Printf("Call failed:Extension %s not found", username)
		return
	}
	targetAddress := &net.UDPAddr{
		IP:   target.IP,
		Port: target.Port,
	}
	activeCalls[callID] = voipinfra.CallSession{
		CallID:     callID,
		CallerAddr: clientAddress,
		CalleeAddr: targetAddress,
	}
	tryingResp := fmt.Sprintf(
		"SIP/2.0 100 Trying\r\n"+
			"Via: %s\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"Call-ID: %s\r\n"+
			"CSeq: %s\r\n"+
			"Content-Length: 0\r\n\r\n",
		via, from, to, callID, cseq,
	)
	conn.WriteToUDP([]byte(tryingResp), clientAddress)
	_, err := conn.WriteToUDP([]byte(rawRequest), targetAddress)
	if err != nil {
		log.Printf("Error forwarding INVITE to %s: %v", username, err)
		return
	}

	log.Printf("Successfully routed INVITE from Caller to %s at %s", username, targetAddress.String())

}
func handleRegister(rawRequest string, registrar *voipinfra.Registrar, conn *net.UDPConn, clientAddress *net.UDPAddr) {
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
