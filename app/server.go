package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Request struct {
	path    string
	headers map[string]string
	body    string
}

type Response struct {
	statusCode  int
	reason      string
	contentType string
	body        string
}

// Router to map paths to handler functions
type Router struct {
	routes map[string]func(*Request) Response
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]func(*Request) Response)}
}

func (r *Router) HandleFunc(method string, path string, handler func(*Request) Response) {
	r.routes[method+path] = handler
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

	method := lineFields[0]
	path := lineFields[1]
	headers := parseHeaders(parts[1:])
	body := parts[len(parts)-1]

	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return
	}

	handler, exists := r.routes[method+"/"+pathParts[1]]
	if !exists {
		writeResponse(conn, 404, "Not Found", "")
		return
	}

	req := Request{path, headers, body}

	res := handler(&req)
	headersToWrite := []httpHeader{
		{"Content-Type", res.contentType},
		{"Content-Length", fmt.Sprintf("%d", len(res.body))},
	}
	handleCompression(&req, &headersToWrite)
	writeResponse(conn, res.statusCode, res.reason, res.body, headersToWrite...)
}

func handleCompression(r *Request, h *[]httpHeader) (bool, string) {
	encoding, exists := r.headers["accept-encoding"]
	if !exists || strings.ToLower(encoding) != "gzip" {
		return false, ""
	}
	*h = append(*h, httpHeader{"Content-Encoding", "gzip"})
	return true, ""
}

func main() {
	router := NewRouter()
	router.HandleFunc("GET", "/", mainPageHandler)
	router.HandleFunc("GET", "/echo", echoHandler)
	router.HandleFunc("GET", "/user-agent", userAgentHandler)
	router.HandleFunc("GET", "/files", filesGetHandler)
	router.HandleFunc("POST", "/files", filesPostHandler)

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

func parseHeaders(headerLines []string) map[string]string {
	headers := make(map[string]string)
	for _, line := range headerLines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(parts[0])] = strings.ToLower(parts[1])
		}
	}
	return headers
}

func echoHandler(r *Request) Response {
	pathParts := strings.Split(r.path, "/")
	if len(pathParts) == 3 {
		return Response{200, "OK", "text/plain", pathParts[2]}
	}
	return Response{404, "Not Found", "text/plain", "Not Found"}
}

func filesGetHandler(r *Request) Response {
	pathParts := strings.Split(r.path, "/")
	if len(pathParts) == 3 {
		var dirPath string
		if len(os.Args) < 3 {
			dirPath = ""
		} else {
			dirPath = os.Args[2]
		}
		fileName := pathParts[2]
		data, err := os.ReadFile(dirPath + fileName)
		if err != nil {
			return Response{404, "Not Found", "text/plain", err.Error()}
		}
		return Response{200, "OK", "application/octet-stream", string(data)}
	}
	return Response{404, "Not Found", "text/plain", "Not Found"}
}

func filesPostHandler(r *Request) Response {
	pathParts := strings.Split(r.path, "/")
	if len(pathParts) == 3 {
		var dirPath string
		if len(os.Args) < 3 {
			dirPath = ""
		} else {
			dirPath = os.Args[2]
		}
		fileName := pathParts[2]
		os.WriteFile(dirPath+fileName, []byte(r.body), 0644)
		return Response{201, "Created", "text/plain", "saved"}
	}
	return Response{404, "Not Found", "text/plain", "Not Found"}
}

func userAgentHandler(r *Request) Response {
	userAgent, exists := r.headers["user-agent"]
	if !exists {
		return Response{400, "Not Found", "text/plain", "User-Agent header not found"}
	}
	return Response{200, "OK", "text/plain", userAgent}
}

func mainPageHandler(_ *Request) Response {
	return Response{200, "OK", "text/html", "<h1>Hello World</h1>"}
}
