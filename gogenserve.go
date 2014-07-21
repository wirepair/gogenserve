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

// GenAddr - a generic address struct which holds
// the necessary information for binding a service
type GenAddr struct {
	Proto string // the protocol (udp, tcp, ws)
	Addr  string // the address (127.0.0.1:8333, :8333)
	Path  string // the path, only used for websockets
}

// GenListener - a generic listener which receives events
// from GenServe as they occur. All events pass the connection
// object which can be used by caller.
type GenListener interface {
	OnConnect(conn *GenConn)                     // notifies that a client has connected, not available in UDP services
	OnRecv(conn *GenConn, data []byte, size int) // notifies that the client has sent data and its size
	OnDisconnect(conn *GenConn)                  // notifies that a client has disconnected
	OnError(conn *GenConn, err error)            // notifies that an error has occurred
	ReadSize() int                               // the number of bytes to read from the socket
}

// GenConn - a generic connection structure holding the transport and the underlying net.Conn
type GenConn struct {
	Transport string
	conn      net.Conn
}

// GenServer - a generic server which offers various protocols to listen on, uses a listener
// to dispatch events as clients connect/send data.
type GenServer interface {
	Listen(addr *GenAddr, listener GenListener)        // given the provided addr information, bind a service and dispatch to listener (TCP/UDP only)
	ListenTCP(addr *GenAddr, listener GenListener)     // dispatch TCP events to listener
	ListenUDP(addr *GenAddr, listener GenListener)     // dispatch UDP events to listener
	MapWSPath(addr *GenAddr, listener GenListener)     // map a websocket path with a listener
	ListenWS(addr *GenAddr)                            // listen on an address
	ListenWSS(addr *GenAddr, certFile, keyFile string) // listen on an secured websocket address
}

// GenServe - implementation of our GenServer
type GenServe struct {
}

// Creates a new generic server, it has no members
func NewGenServe() *GenServe {
	return &GenServe{}
}

// Listen - validates that the GenAddr is in correct form for TCP or UDP and listens on
// the given address
func (g *GenServe) Listen(addr *GenAddr, listener GenListener) {
	if addr.Proto == "" || addr.Addr == "" {
		log.Fatal("Protocol or Address invalid for Listen.")
	}

	if strings.HasPrefix(addr.Proto, "tcp") {
		g.listenTCP(addr, listener)
	} else {
		g.listenUDP(addr, listener)
	}
}

// ListenUDP - Listens on a given network/port provided by addr and dispatches UDP events to the listener.
func (g *GenServe) ListenUDP(addr *GenAddr, listener GenListener) {
	if addr.Proto == "" {
		addr.Proto = "udp"
	}

	if !strings.HasPrefix(addr.Proto, "udp") {
		log.Fatal("Invalid protocol set for ListenUDP")
	}
	g.listenUDP(addr, listener)
}

// listenUDP - resolves / validates the protocol and address, binds to the network and accepts connections
// reads are put in their own goroutines.
func (g *GenServe) listenUDP(addr *GenAddr, listener GenListener) {
	if addr.Addr == "" {
		log.Fatal("Address not set for UDP server")
	}

	udp, err := net.ResolveUDPAddr(addr.Proto, addr.Addr)
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

// ListenTCP - Listens on a given network/port provided by addr and dispatches TCP events to the listener.
func (g *GenServe) ListenTCP(addr *GenAddr, listener GenListener) {
	if addr.Proto == "" {
		addr.Proto = "tcp"
	}

	if !strings.HasPrefix(addr.Proto, "tcp") {
		log.Fatal("Invalid protocol set for ListenTCP")
	}
	g.listenTCP(addr, listener)
}

// listenUDP - validates the protocol and address, binds to the network and accepts connections, in its own
// go routine as well as reads put in their own go routines. Events dispatched to the listener.
func (g *GenServe) listenTCP(addr *GenAddr, listener GenListener) {
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
				conn := &GenConn{Transport: addr.Proto}
				listener.OnError(conn, err)
				continue
			}
			newConn := &GenConn{Transport: addr.Proto, conn: c}
			listener.OnConnect(newConn)
			// read loop
			go read(listener, newConn)
		}
	}()
}

// MapWSPath - Maps a websocket path to a listener for dispatching events to. Should
// be called prior to calling ListenWS or ListenWSS
// It is possible to map the same listener to multiple paths, provided the passed
// in addr is has a different Path defined.
func (g *GenServe) MapWSPath(addr *GenAddr, listener GenListener) {
	http.Handle(addr.Path, websocket.Handler(func(ws *websocket.Conn) {
		webSocketHandler(ws, listener)
	}))
}

// ListenWS - Listens on the given address for WebSocket connections, MapWSPath should be called first
func (g *GenServe) ListenWS(addr *GenAddr) {
	go func() {
		err := http.ListenAndServe(addr.Addr, nil)
		if err != nil {
			log.Fatalf("error listening on %s: %v", addr.Addr, err)
		}
	}()
}

// ListenWS - Listens on the given address for secured WebSocket connections, MapWSPath should be called first
func (g *GenServe) ListenWSS(addr *GenAddr, certFile, keyFile string) {
	go func() {
		err := http.ListenAndServeTLS(addr.Addr, certFile, keyFile, nil)
		if err != nil {
			log.Fatalf("error listening on %s: %v", addr.Addr, err)
		}
	}()
}

// webSocketHandler - dispatches OnConnect events when new clients connect, reads are run in their own
// go routines. Once IsServerConn returns false, the connection is considered dead and an OnDisconnect
// event is dispatched.
func webSocketHandler(ws *websocket.Conn, listener GenListener) {
	newConn := &GenConn{Transport: "websocket", conn: ws}
	listener.OnConnect(newConn)
	for ws.IsServerConn() {
		go read(listener, newConn)
	}
	listener.OnDisconnect(newConn)
}

// read - Reads bytes from theconnection and dispatches OnRecv events. If io.EOF is returned
// by the underlying read, the connection is considered dead and an OnDisconnect event is dispatched.
func read(listener GenListener, conn *GenConn) {
	data := bytes.NewBuffer(nil)
	msg := make([]byte, listener.ReadSize())
	for {
		n, err := conn.conn.Read(msg[0:])
		if err == io.EOF {
			listener.OnDisconnect(conn)
		}
		if err != nil {
			log.Printf("error %v\n", err)
			listener.OnError(conn, err)
			return
		}
		data.Write(msg[0:n])
		listener.OnRecv(conn, data.Bytes(), len(data.Bytes()))
		data.Reset()
	}
}
