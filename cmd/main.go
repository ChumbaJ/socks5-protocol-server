package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	RepSuccess          = byte(0x00)
	RepFailure          = byte(0x01)
	RepNotAllowed       = byte(0x02)
	RepNetUnreachable   = byte(0x03)
	RepHostUnreachable  = byte(0x04)
	RepRefused          = byte(0x05)
	RepTTLExpired       = byte(0x06)
	RepCmdNotSupported  = byte(0x07)
	RepAddrNotSupported = byte(0x08)
)

func serverReply(conn net.Conn, rep byte) {
	conn.Write([]byte{0x05, rep, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func handleConn(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered from panic:", r)
		}
	}()
	defer conn.Close()

	// read greeting from client
	buf := make([]byte, 2)
	io.ReadFull(conn, buf)

	version := buf[0]
	if version != 0x05 {
		return // not socks5
	}

	nmethods := buf[1]
	methods := make([]byte, nmethods)
	io.ReadFull(conn, methods)

	// select auth method - no auth 00
	conn.Write([]byte{0x05, 0x00})

	// read request from client
	req := make([]byte, 4)
	io.ReadFull(conn, req)

	atyp := req[3]

	var host string
	var port uint16

	switch atyp {
	case 0x01:
		addr := make([]byte, 4)
		io.ReadFull(conn, addr)

		host = net.IP(addr).String()
	case 0x03:
		addrLen := make([]byte, 1)
		io.ReadFull(conn, addrLen)

		domain := make([]byte, addrLen[0])
		io.ReadFull(conn, domain)

		host = string(domain)
	case 0x04:
		addr := make([]byte, 16)
		io.ReadFull(conn, addr)

		host = net.IP(addr).String()
	default:
		return
	}

	portBytes := make([]byte, 2)
	io.ReadFull(conn, portBytes)
	port = binary.BigEndian.Uint16(portBytes)

	// conn to target
	address := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("connecting to: %s\n", address)

	targetConn, err := net.Dial("tcp", address)
	if err != nil {
		serverReply(conn, RepFailure)
		return
	}
	defer targetConn.Close()

	// send SUCCESS to client
	serverReply(conn, RepSuccess)

	// run bidirectional relay

	errChan := make(chan error, 2)
	go func() {
		_, err := io.Copy(targetConn, conn)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(conn, targetConn)
		errChan <- err
	}()

	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			fmt.Printf("relay error: %s\n", err.Error())
		}
	}
}

func main() {
	listener, err := net.Listen("tcp4", ":1081")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer listener.Close()

	fmt.Println("server is listening on port 1081")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err.Error())
		}

		go handleConn(conn)

	}
}
