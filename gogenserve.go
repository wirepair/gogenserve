package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"log"
	"net"
	"net/http"
)

type GenHandler func(conn *GenConn) error

type GenConn struct {
	Transport string
	conn      net.Conn
	Err       error
}

func (g *GenConn) Send(data []byte) (int, error) {
	return g.conn.Write(data)
}

func (g *GenConn) Recv(data []byte) (net.Addr, error) {
	return g.Recv(data)
}

type GenServer interface {
	Listen(proto, addr string, conn chan<- *GenConn) error
	ListenTCP(addr string, conn chan<- *GenConn) error
	ListenUDP(addr string, conn chan<- *GenConn) error
	ListenWS(addr, path string, conn chan<- *GenConn) error
}

type GenServe struct {
}

func NewGenServe() *GenServe {
	return &GenServe{}
}

func (g *GenServe) ListenTCP(addr string, conn chan<- *GenConn) error {
	return g.Listen("tcp", addr, conn)
}

func (g *GenServe) ListenUDP(addr string, conn chan<- *GenConn) error {
	return g.Listen("udp", addr, conn)
}

func (g *GenServe) Listen(proto, addr string, conn chan<- *GenConn) error {
	ln, err := net.Listen(proto, addr)
	if err != nil {
		log.Fatalf("Error listening on %s socket at %s, %v", proto, addr, err)
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				newConn := &GenConn{Err: err}
				conn <- newConn
				continue
			}
			newConn := &GenConn{Transport: proto, conn: c}
			fmt.Println("Dispatching...")
			conn <- newConn
		}
	}()
	return nil
}

func (g *GenServe) ListenWS(addr, path string, conn chan<- *GenConn) error {
	http.Handle(path, websocket.Handler(func(ws *websocket.Conn) {
		newConn := &GenConn{Transport: "websocket", conn: ws}
		conn <- newConn
	}))
	go http.ListenAndServe(addr, nil)
	return nil
}
