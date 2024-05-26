package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}

type httpStatus struct {
	version string
	code    int
	reason  string
}

func (s *httpStatus) String() string {
	return fmt.Sprintf("%s %d %s", s.version, s.code, s.reason)
}

type httpHeader []struct {
	name  string
	value string
}

func (h *httpHeader) String() string {
	sb := new(strings.Builder)
	for _, v := range *h {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", v.name, v.value))
	}
	return sb.String()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	status := httpStatus{
		version: "HTTP/1.1",
		code:    200,
		reason:  "OK",
	}
	header := httpHeader{}

	sb := strings.Builder{}
	sb.WriteString(status.String())
	sb.WriteString("\r\n")
	sb.WriteString(header.String())
	sb.WriteString("\r\n")

	conn.Write([]byte(sb.String()))
}
