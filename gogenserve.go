package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	//"fmt"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type GenAddr struct {
	Proto string
	Addr  string
	Path  string
}

type GenListener interface {
	OnConnect(conn *GenConn)
	OnRecv(conn *GenConn, data []byte, size int)
	OnDisconnect(conn *GenConn)
	OnError(conn *GenConn, err error)
	Addr() *GenAddr
}

type GenConn struct {
	Transport string
	conn      net.Conn
	Err       error
}

type GenServer interface {
	Listen(listener GenListener)
	ListenTCP(listener GenListener)
	ListenUDP(listener GenListener)
	MapWSPath(listener GenListener)
	ListenWS(listener GenListener)
	ListenWSS(listener GenListener, certFile, keyFile string)
}

type GenServe struct {
}

func NewGenServe() *GenServe {
	return &GenServe{}
}

func (g *GenServe) Listen(listener GenListener) {
	addr := listener.Addr()
	if addr.Proto == "" || addr.Addr == "" {
		log.Fatal("Protocol or Address invalid for Listen.")
	}

	if strings.HasPrefix(addr.Proto, "tcp") {
		g.listenTCP(listener)
	} else {
		g.listenUDP(listener)
	}
}

func (g *GenServe) ListenUDP(listener GenListener) {
	if listener.Addr().Proto == "" {
		listener.Addr().Proto = "udp"
	}

	if !strings.HasPrefix(listener.Addr().Proto, "udp") {
		log.Fatal("Invalid protocol set for ListenUDP")
	}
	g.listenUDP(listener)
}

func (g *GenServe) listenUDP(listener GenListener) {
	addr := listener.Addr()
	if addr.Addr == "" {
		log.Fatal("Address not set for UDP server")
	}

	udp, err := net.ResolveUDPAddr(addr.Addr, addr.Proto)
	if err != nil {
		log.Fatalf("Error in resolve udp address: %v\n", err)
	}
	ln, err := net.ListenUDP(addr.Proto, udp)
	if err != nil {
		log.Fatalf("Error listening on %s socket at %s, %v\n", addr.Proto, addr.Addr, err)
	}
	newConn := &GenConn{Transport: addr.Proto, conn: ln}
	go func() {
		for {
			read(listener, newConn)
		}
	}()
}

func (g *GenServe) ListenTCP(listener GenListener) {
	if listener.Addr().Proto == "" {
		listener.Addr().Proto = "tcp"
	}

	if !strings.HasPrefix(listener.Addr().Proto, "tcp") {
		log.Fatal("Invalid protocol set for ListenTCP")
	}
	g.listenTCP(listener)
}

func (g *GenServe) listenTCP(listener GenListener) {
	addr := listener.Addr()
	if addr.Addr == "" {
		log.Fatal("Address not set for UDP server")
	}

	ln, err := net.Listen(addr.Proto, addr.Addr)
	if err != nil {
		log.Fatalf("Error listening on %s socket at %s, %v\n", addr.Proto, addr.Addr, err)
	}
	// accept loop
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				conn := &GenConn{Transport: addr.Proto, Err: err}
				listener.OnError(conn, err)
				continue
			}
			newConn := &GenConn{Transport: addr.Proto, conn: c}
			listener.OnConnect(newConn)
			// read loop
			go func() {
				read(listener, newConn)
			}()
		}
	}()
}

func (g *GenServe) MapWSPath(listener GenListener) {
	http.Handle(listener.Addr().Path, websocket.Handler(func(ws *websocket.Conn) {
		webSocketHandler(ws, listener)
	}))
}

func (g *GenServe) ListenWS(listener GenListener) {
	go func() {
		err := http.ListenAndServe(listener.Addr().Addr, nil)
		if err != nil {
			log.Fatalf("error listening on %s: %v", listener.Addr().Addr, err)
		}
	}()
}

func (g *GenServe) ListenWSS(listener GenListener, certFile, keyFile string) {
	go func() {
		err := http.ListenAndServeTLS(listener.Addr().Addr, certFile, keyFile, nil)
		if err != nil {
			log.Fatalf("error listening on %s: %v", listener.Addr().Addr, err)
		}
	}()
}

func webSocketHandler(ws *websocket.Conn, listener GenListener) {
	newConn := &GenConn{Transport: "websocket", conn: ws}
	listener.OnConnect(newConn)
	for ws.IsClientConn() && ws.IsServerConn() {
		read(listener, newConn)
	}
	listener.OnDisconnect(newConn)
}

func read(listener GenListener, conn *GenConn) {
	data := bytes.NewBuffer(nil)
	var msg [512]byte
	for {
		log.Printf("about to read")
		n, err := conn.conn.Read(msg[0:])
		if err != nil && err != io.EOF {
			log.Printf("error %v\n", err)
			listener.OnError(conn, err)
			return
		}
		data.Write(msg[0:n])
		log.Printf("done reading")
		log.Printf("%v\n", err)
		if err == io.EOF {
			listener.OnRecv(conn, data.Bytes(), len(data.Bytes()))
			data.Reset()
		}
	}
}
