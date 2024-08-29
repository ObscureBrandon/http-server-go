package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const CRLF = "\r\n"

type Headers map[string]string

func (h Headers) Get(key string) string {
	if val, ok := h[key]; ok {
		return val
	}

	return ""
}

func (h Headers) Set(key string, value string) {
	if value == "" {
		delete(h, key)
		return
	}

	h[key] = value
}

type Params map[string]string

func (p Params) Get(key string) string {
	if val, ok := p[key]; ok {
		return val
	}

	return ""
}

type Request struct {
	Method   string
	Path     []string
	FullPath string
	Headers  Headers
	Params   Params
	Body     string
}

type Response struct {
	StatusCode int
	Status     string
	Headers    Headers
	Body       string
}

type Context struct {
	Request *Request
}

func (c *Context) text(statusCode int, body string) Response {
	resp := Response{StatusCode: statusCode, Status: http.StatusText(statusCode), Body: body, Headers: Headers{}}
	resp.Headers.Set("Content-Type", "text/plain")
	resp.Headers.Set("Content-Length", strconv.Itoa(len(body)))

	return resp
}

func (r *Response) Bytes() []byte {
	var out bytes.Buffer

	out.WriteString(fmt.Sprintf("HTTP/1.1 %d %s", r.StatusCode, r.Status))
	out.WriteString(CRLF)
	for k, v := range r.Headers {
		out.WriteString(fmt.Sprintf("%s: %s", k, v))
		out.WriteString(CRLF)
	}
	out.WriteString(CRLF)
	out.WriteString(r.Body)

	return out.Bytes()
}

type Route struct {
	allowedMethod string
	handler       func(Context) Response
}

type Router struct {
	routes map[string]Route
}

func (r *Router) resolvePath(request *Request) (Route, bool) {
	if route, ok := r.routes[request.Method+" "+request.FullPath]; ok {
		return route, true
	}

	for route_path, route := range r.routes {
		if !strings.HasPrefix(route_path, request.Method+" ") {
			continue
		}

		without_method := strings.TrimPrefix(route_path, request.Method+" ")
		sep_route_path := strings.Split(without_method, "/")[1:]

		if len(sep_route_path) > len(request.Path) {
			continue
		}

		found := true
		for i, route_part := range sep_route_path {
			if route_part == request.Path[i] {
				continue
			}

			if strings.HasPrefix(route_part, ":") {
				request.Params[strings.TrimPrefix(route_part, ":")] = request.Path[i]
			} else {
				found = false
				break
			}
		}

		if found {
			return route, true
		}
	}

	return Route{}, false
}

func (r *Router) GET(path string, handler func(Context) Response) {
	r.routes[http.MethodGet+" "+path] = Route{allowedMethod: http.MethodGet, handler: handler}
}

func (r *Router) POST(path string, handler func(Context) Response) {
	r.routes[http.MethodPost+" "+path] = Route{allowedMethod: http.MethodPost, handler: handler}
}

func (r *Router) PUT(path string, handler func(Context) Response) {
	r.routes[http.MethodPut+" "+path] = Route{allowedMethod: http.MethodPut, handler: handler}
}

func (r *Router) HEAD(path string, handler func(Context) Response) {
	r.routes[http.MethodHead+" "+path] = Route{allowedMethod: http.MethodHead, handler: handler}
}

func (r *Router) DELETE(path string, handler func(Context) Response) {
	r.routes[http.MethodDelete+" "+path] = Route{allowedMethod: http.MethodDelete, handler: handler}
}

func (r *Router) PATCH(path string, handler func(Context) Response) {
	r.routes[http.MethodPatch+" "+path] = Route{allowedMethod: http.MethodPatch, handler: handler}
}

func NewRouter() *Router {
	return &Router{routes: map[string]Route{}}
}

func (r *Router) Start(port int) {
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d", port)
		os.Exit(1)
	}

	fmt.Printf("Listening on port %d\n", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go r.handleConnection(conn)
	}
}

func (r *Router) handleConnection(conn net.Conn) {
	defer conn.Close()

	buff := make([]byte, 1024)
	conn.Read(buff)

	str_buff := string(buff)

	head := strings.Split(str_buff, CRLF+CRLF)[0]
	body := strings.Split(str_buff, CRLF+CRLF)[1]

	request_line := strings.SplitAfterN(head, CRLF, 2)[0]
	headers := strings.SplitAfterN(head, CRLF, 2)[1]

	method := strings.Split(request_line, " ")[0]
	path := strings.Split(request_line, " ")[1]
	sep_path := strings.Split(path, "/")[1:]

	req := Request{Method: method, FullPath: path, Path: sep_path, Headers: Headers{}, Params: Params{}, Body: body}
	for _, header := range strings.Split(headers, CRLF) {
		idx := strings.Index(header, ":")
		req.Headers.Set(header[:idx], header[idx+2:])
	}

	ctx := Context{Request: req}
	route, ok := r.resolvePath(req)
	if !ok {
		resp := ctx.text(http.StatusNotFound, "")
		conn.Write(resp.Bytes())
		return
	}

	resp := route.handler(ctx)
	conn.Write(resp.Bytes())
}

func main() {
	router := NewRouter()

	router.GET("/echo/:msg/:meow", func(ctx Context) Response {
		return ctx.text(http.StatusOK, ctx.Request.Params.Get("msg"))
	})

	router.GET("/user-agent", func(ctx Context) Response {
		return ctx.text(http.StatusOK, ctx.Request.Headers.Get("User-Agent"))
	})

	router.Start(4221)
}
