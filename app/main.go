package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

const (
	httpVersion = "HTTP/1.1"
)

func main() {

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
			} else {
				n, err := w.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
				if err != nil {
					fmt.Println(n, err)
				}
				w.Flush()
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
