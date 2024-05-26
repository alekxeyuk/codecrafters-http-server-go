package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Router to map paths to handler functions
type Router struct {
	routes map[string]func(string) (int, string, string)
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]func(string) (int, string, string))}
}

func (r *Router) HandleFunc(path string, handler func(string) (int, string, string)) {
	r.routes[path] = handler
}

func (r *Router) ServeHTTP(conn net.Conn, request string) {
	parts := strings.Split(request, "\r\n")
	if len(parts) < 1 {
		return
	}
	lineFields := strings.Fields(parts[0])
	if len(lineFields) < 2 {
		return
	}

	path := lineFields[1]
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return
	}

	handler, exists := r.routes["/"+pathParts[1]]
	if !exists {
		writeResponse(conn, 404, "Not Found", "")
		return
	}

	statusCode, contentType, body := handler(path)
	headers := []httpHeader{
		{"Content-Type", contentType},
		{"Content-Length", fmt.Sprintf("%d", len(body))},
	}
	writeResponse(conn, statusCode, "OK", body, headers...)
}

func main() {
	router := NewRouter()
	router.HandleFunc("/echo", echoHandler)

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
			continue
		}
		go handleConnection(conn, router)
	}
}

func handleConnection(conn net.Conn, router *Router) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}

	request := string(buf[:n])
	router.ServeHTTP(conn, request)
}

type httpHeader struct {
	name  string
	value string
}

func (h *httpHeader) String() string {
	return fmt.Sprintf("%s: %s\r\n", h.name, h.value)
}

func writeResponse(conn net.Conn, statusCode int, statusReason, body string, headers ...httpHeader) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusReason))
	for _, header := range headers {
		sb.WriteString(header.String())
	}
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")
	conn.Write([]byte(sb.String()))
}

func echoHandler(path string) (int, string, string) {
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 3 {
		return 200, "text/plain", pathParts[2]
	}
	return 404, "text/plain", "Not Found"
}
