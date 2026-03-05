package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

func handleConn(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered from panic:", r)
		}
	}()

	defer conn.Close()

	fmt.Println("new conn:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	req, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println(err.Error())
	}

	if req.Method != http.MethodConnect {
		return
	}

	target, err := net.Dial("tcp4", req.Host)
	if err != nil {
		fmt.Println("dial error", err.Error())
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer target.Close()

	conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(target, conn)
		errCh <- err
	}()

	go func() {
		_, err = io.Copy(conn, target)
		errCh <- err
	}()

	for err := range errCh {
		if err != nil {
			fmt.Println(err.Error())
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
