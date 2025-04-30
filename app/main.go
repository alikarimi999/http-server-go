package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

const (
	httpVersion = "HTTP/1.1"
)

var baseDirectory = ""

type Args struct {
	Directory string `arg:"-d,--directory" default:"." help:"Set root directory for serving files"`
}

func main() {
	args := Args{}
	arg.MustParse(&args)
	baseDirectory = args.Directory

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		panic(err)
	}

	s := Server{
		listener: l,
	}
	s.Start()
}

type Server struct {
	listener net.Listener
}

func (s *Server) Start() {
	fmt.Printf("server started on %s - %s\n", s.listener.Addr().Network(), s.listener.Addr().String())
	defer s.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
		}
		go s.handle(conn)
	}
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("connection from %v\n", conn.RemoteAddr().String())

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	for {
		method, path, version, err := parseRequestLine(r)
		if err != nil {
			fmt.Println("parse request line error:", err)
			return
		}

		if version != httpVersion {
			fmt.Println("unsupported http version:", version)
			return
		}

		header, err := parseHeader(r)
		if err != nil {
			fmt.Println("parse header error:", err)
			return
		}

		fmt.Printf("request: '%s %s %s'\n", method, path, version)
		fmt.Println("header: ", header)
		if method == "GET" {
			if path == "/" {
				n, err := w.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
				if err != nil {
					fmt.Println(n, err)
				}
				w.Flush()
			} else if strings.HasPrefix(path, "/files/") {
				fileName := strings.SplitAfter(path, "/files/")[1]
				msg, err := os.ReadFile(baseDirectory + fileName)
				if err != nil {
					res := NewResponse(version, http.StatusNotFound, "")
					res.Write(w)
					return
				}

				res := NewResponse(version, http.StatusOK, string(msg))
				res.SetHeader("Content-Type", "application/octet-stream")
				res.SetHeader("Content-Length", fmt.Sprintf("%d", len(msg)))
				if err := res.Write(w); err != nil {
					fmt.Println(err)
				}

			} else if strings.HasPrefix(path, "/user-agent") {
				userAgent := header["User-Agent"]
				res := NewResponse(version, http.StatusOK, userAgent)
				res.SetHeader("Content-Type", "text/plain")
				res.SetHeader("Content-Length", fmt.Sprintf("%d", len(userAgent)))
				if err := res.Write(w); err != nil {
					fmt.Println(err)
				}

			} else if strings.HasPrefix(path, "/echo/") {
				msg := strings.SplitAfter(path, "/echo/")[1]

				res := NewResponse(version, http.StatusOK, msg)
				res.SetHeader("Content-Type", "text/plain")
				res.SetHeader("Content-Length", fmt.Sprintf("%d", len(msg)))

				if err := res.Write(w); err != nil {
					fmt.Println(err)
				}

			} else {
				n, err := w.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				if err != nil {
					fmt.Println(n, err)
				}
				w.Flush()
			}
		} else if method == "POST" {
			if strings.HasPrefix(path, "/files/") {
				fileName := strings.SplitAfter(path, "/files/")[1]
				body, err := parseBody(r, header)
				if err != nil {
					fmt.Println(err)
					return
				}

				err = os.MkdirAll(baseDirectory, 0755)
				if err != nil {
					fmt.Println(err)
					return
				}
				if err := os.WriteFile(baseDirectory+fileName, []byte(body), 0644); err != nil {
					fmt.Println(err)
					return
				}
				res := NewResponse(version, http.StatusCreated, "")
				res.Write(w)

			}
		}
	}
}

func parseHeader(r *bufio.Reader) (map[string]string, error) {
	header := make(map[string]string)

	var line []byte
	for {
		l, isPrefix, err := r.ReadLine()
		if err != nil {
			break
		}

		if len(l) == 0 {
			// End of headers
			break
		}

		if isPrefix {
			line = append(line, l...)
			continue
		}

		line = append(line, l...)

		parts := strings.Split(string(line), ": ")
		if len(parts) == 2 {
			key := strings.SplitAfter(parts[0], ":")[0]
			header[key] = parts[1]
		}

		line = []byte{}
	}

	return header, nil
}

func parseBody(r *bufio.Reader, headers map[string]string) (string, error) {
	lengthStr, ok := headers["Content-Length"]
	if !ok {
		return "", fmt.Errorf("missing Content-Length header")
	}

	var length int
	_, err := fmt.Sscanf(lengthStr, "%d", &length)
	if err != nil {
		return "", fmt.Errorf("invalid Content-Length: %v", err)
	}

	body := make([]byte, length)
	_, err = r.Read(body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseRequestLine(r *bufio.Reader) (string, string, string, error) {
	line, _, err := r.ReadLine()
	if err != nil {
		return "", "", "", err
	}

	parts := strings.Split(string(line), " ")
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid request line: %s", line)
}

type Response struct {
	version    string
	statusCode int
	header     map[string]string
	body       string
}

func NewResponse(version string, statusCode int, body string) *Response {
	return &Response{
		version:    version,
		statusCode: statusCode,
		body:       body,
	}
}

func (r *Response) SetHeader(key, value string) {
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header[key] = value
}

func (r *Response) Write(w *bufio.Writer) error {
	var res string
	responseLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.statusCode, http.StatusText(r.statusCode))
	res += responseLine

	for key, value := range r.header {
		res += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	res += "\r\n"
	res += r.body

	_, err := w.Write([]byte(res))
	if err != nil {
		return err
	}

	return w.Flush()
}
