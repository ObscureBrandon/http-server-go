package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

type Headers map[string]string

func (h Headers) Get(key string) string {
	return h[key]
}

func (h Headers) Set(key string, value string) {
	h[key] = value
}

type Request struct {
	Method  string
	Path    []string
	Headers Headers
	Body    string
}

type Response struct {
	StatusCode    int
	Status        string
	ContentType   string
	ContentLength int
	Body          string
}

func (r *Response) Bytes() []byte {
	var out bytes.Buffer

	out.WriteString(fmt.Sprintf("HTTP/1.1 %d %s", r.StatusCode, r.Status))
	out.WriteString(delimiter)
	out.WriteString(fmt.Sprintf("Content-Type: %s", r.ContentType))
	out.WriteString(delimiter)
	out.WriteString(fmt.Sprintf("Content-Length: %d", r.ContentLength))
	out.WriteString(delimiter)
	out.WriteString(delimiter)
	out.WriteString(r.Body)

	return out.Bytes()
}

const delimiter = "\r\n"

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buff := make([]byte, 1024)
	conn.Read(buff)

	str_buff := string(buff)

	head := strings.Split(str_buff, delimiter+delimiter)[0]
	body := strings.Split(str_buff, delimiter+delimiter)[1]

	request_line := strings.SplitAfterN(head, delimiter, 2)[0]
	headers := strings.SplitAfterN(head, delimiter, 2)[1]

	method := strings.Split(request_line, " ")[0]
	path := strings.Split(request_line, " ")[1]
	sep_path := strings.Split(path, "/")[1:]

	req := Request{Method: method, Path: sep_path, Headers: Headers{}, Body: body}
	for _, header := range strings.Split(headers, delimiter) {
		idx := strings.Index(header, ":")
		req.Headers.Set(header[:idx], header[idx+1:])
	}

	if req.Path[0] != "echo" && req.Path[0] != "user-agent" {
		resp := Response{StatusCode: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound), ContentType: "text/plain", ContentLength: 0, Body: ""}
		conn.Write(resp.Bytes())
		conn.Close()
		return
	}

	if req.Path[0] == "echo" {
		resp := Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), ContentType: "text/plain", ContentLength: len(req.Path[1]), Body: req.Path[1]}
		conn.Write(resp.Bytes())
		conn.Close()
		return
	}

	agent := req.Headers.Get("User-Agent")
	resp := Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), ContentType: "text/plain", ContentLength: len(agent), Body: agent}
	conn.Write(resp.Bytes())
	conn.Close()
}
