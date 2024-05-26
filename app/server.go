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

type httpHeader struct {
	name  string
	value string
}

type httpHeaders []httpHeader

func (h *httpHeader) String() string {
	return fmt.Sprintf("%s: %s\r\n", h.name, h.value)
}

func (h *httpHeaders) String() string {
	sb := new(strings.Builder)
	for _, v := range *h {
		sb.WriteString(v.String())
	}
	return sb.String()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	responseStatus := httpStatus{
		version: "HTTP/1.1",
		code:    200,
		reason:  "OK",
	}

	header := httpHeaders{}

	var body string

	buf := make([]byte, 1024)
	conn.Read(buf)
	parts := strings.Split(string(buf), "\r\n")
	requestSize := len(parts)
	if requestSize > 1 {
		lineFields := strings.Fields(parts[0])

		pathSplit := strings.Split(lineFields[1], "/")

		if len(pathSplit) == 3 && pathSplit[1] == "echo" {
			header = append(header, httpHeader{"Content-Type", "text/plain"})
			body = pathSplit[2]
			header = append(header, httpHeader{"Content-Length", fmt.Sprintf("%d", len([]byte(body)))})
		}

		//if lineFields[1] = "/" {
		//	responseStatus.code = 404
		//	responseStatus.reason = "Not Found"
		//}
	}

	sb := strings.Builder{}
	sb.WriteString(responseStatus.String())
	sb.WriteString("\r\n")
	sb.WriteString(header.String())
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")

	conn.Write([]byte(sb.String()))
}
