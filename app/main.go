package main

import (
	"fmt"
	"net"
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
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("read error:", err)
			return
		}

		fmt.Printf("received: %s\n", buf[:n])
		resp := fmt.Sprintf("response to '%s'\n", buf[:n])
		_, err = conn.Write([]byte(resp))
		if err != nil {
			fmt.Println("write error:", err)
		}
	}
}
