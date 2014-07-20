package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	//"fmt"
	"log"
	"net"
	"net/http"
	"strings"
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
	Listen(proto, addr string, conn chan<- *GenConn)
	ListenTCP(addr string, conn chan<- *GenConn)
	ListenUDP(addr string, conn chan<- *GenConn)
	MapWSPath(path string, conn chan<- *GenConn)
	ListenWS(addr, path string, conn chan<- *GenConn)
	ListenWSS(addr, path, certFile, keyFile, conn chan<- *GenConn)
	OnConnect(conn chan<- *GenConn)
	OnError(conn chan<- *GenConn)
	OnDisconnect(conn chan<- *GenConn)
	OnSent(conn chan<- *GenConn)
	OnRecv(conn chan<- *GenConn)
}

type GenServe struct {
}

func NewGenServe() *GenServe {
	return &GenServe{}
}

func (g *GenServe) Listen(proto, addr string, conn chan<- *GenConn) {
	if strings.HasPrefix(proto, "tcp") {
		g.listenTCP(proto, addr, conn)
	} else {
		g.listenUDP(proto, addr, conn)
	}
}

func (g *GenServe) ListenUDP(addr string, conn chan<- *GenConn) {
	g.listenUDP("udp", addr, conn)
}

func (g *GenServe) listenUDP(proto, addr, conn chan<- *GenConn) {
	udp := net.ResolveUDPAddr(proto, addr)
	ln, err := net.ListenUDP(proto, udp)
	if err != nil {
		log.Fatalf("Error listenong on %s socket at %s, %v", proto, addr, err)
	}

}

func (g *GenServe) ListenTCP(addr string, conn chan<- *GenConn) {
	g.listenTCP("tcp", addr, conn)
}

func (g *GenServe) listenTCP(proto, addr string, conn chan<- *GenConn) {
	ln, err := net.Listen(proto, addr)
	if err != nil {
		log.Fatalf("Error listening on %s socket at %s, %v", proto, addr, err)
	}

	go func(conn chan<- *GenConn) {
		for {
			c, err := ln.Accept()
			if err != nil {
				conn <- &GenConn{Transport: proto, Err: err}
				continue
			}
			conn <- &GenConn{Transport: proto, conn: c}
		}
	}(conn)
}

func (g *GenServe) MapWSPath(path string, conn chan<- *GenConn) {
	http.Handle(path, websocket.Handler(func(ws *websocket.Conn) {
		webSocketHandler(ws, conn)
	}))
}

func (g *GenServe) ListenWS(addr string) {
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatalf("error listening on %s: %v", addr, err)
		}
	}()
}

func (g *GenServe) ListenWSS(addr, certFile, keyFile string, conn chan<- *GenConn) {
	go func() {
		err := http.ListenAndServeTLS(addr, certFile, keyFile, nil)
		if err != nil {
			conn <- &GenConn{Err: err}
			return
		}
	}()
}

func webSocketHandler(ws *websocket.Conn, conn chan<- *GenConn) {
	newConn := &GenConn{Transport: "websocket", conn: ws}
	conn <- newConn
}
