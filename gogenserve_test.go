package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net"
	"sync"
	"testing"
)

type DefaultListener struct {
	wg   *sync.WaitGroup
	addr *GenAddr
}

func (d *DefaultListener) OnConnect(conn *GenConn) {
	fmt.Printf("Yup got connection %v\n", conn)
	d.wg.Done()
}

func (d *DefaultListener) OnDisconnect(conn *GenConn) {
	fmt.Printf("Yup connection dropped %v\n", conn)
}

func (d *DefaultListener) OnRecv(conn *GenConn, data []byte, size int) {
	fmt.Printf("Yup got data %s size: %d from connection %v\n", data[0:size], size, conn)
	d.wg.Done()
}

func (d *DefaultListener) OnError(conn *GenConn, err error) {
	fmt.Printf("Yup got error on connection %v: %v\n", conn, err)
}

func (d *DefaultListener) ReadSize() int {
	return 8196
}

func NewListener() *DefaultListener {
	l := new(DefaultListener)
	l.wg = new(sync.WaitGroup)
	l.wg.Add(1) // for connection
	return l
}

func TestNewTcpServer(t *testing.T) {
	listener := NewListener()
	addr := &GenAddr{Proto: "tcp", Addr: ":8333", Path: ""}
	listener.wg.Add(1)
	serve := NewGenServe()
	serve.ListenTCP(addr, listener)
	conn := tcpConnection("localhost:8333", t)
	conn.Write([]byte("zooops"))
	conn.Close()
	listener.wg.Wait()
}

func TestNewWSServer(t *testing.T) {
	listener := NewListener()
	addr := &GenAddr{Proto: "ws", Addr: ":8333", Path: "/bonk"}
	listener.wg.Add(1)
	serve := NewGenServe()
	serve.MapWSPath(addr, listener)
	serve.ListenWS(addr.Addr)
	conn := wsConnection("localhost:8333", "/bonk", t)
	_, err := conn.Write([]byte("zooops"))
	if err != nil {
		t.Fatalf("err not nil: %v\n", err)
	}
	listener.wg.Wait()
}

func TestNewUDPServer(t *testing.T) {
	listener := NewListener()
	addr := &GenAddr{Proto: "udp", Addr: ":8333", Path: ""}
	serve := NewGenServe()
	serve.ListenUDP(addr, listener)
	conn := udpConnection("localhost:8333", t)
	conn.Write([]byte("zooops"))
	listener.wg.Wait()
}

func TestMultiUDPClients(t *testing.T) {
	listener := NewListener()
	addr := &GenAddr{Proto: "udp", Addr: ":8333", Path: ""}
	serve := NewGenServe()
	listener.wg.Add(4)
	serve.ListenUDP(addr, listener)
	for i := 0; i < 5; i++ {
		conn := udpConnection("localhost:8333", t)
		conn.Write([]byte(fmt.Sprintf("zoops %d\n", i)))
	}
	listener.wg.Wait()
}

func TestMultiWSPathsClients(t *testing.T) {
	listener := NewListener()
	addrBonk := &GenAddr{Proto: "ws", Addr: ":8333", Path: "/bonk"}
	addrDonk := &GenAddr{Proto: "ws", Addr: ":8333", Path: "/donk"}
	listener.wg.Add(3)
	serve := NewGenServe()
	serve.MapWSPath(addrBonk, listener)
	serve.MapWSPath(addrDonk, listener)
	serve.ListenWS(addrBonk.Addr)
	conn := wsConnection("localhost:8333", "/bonk", t)
	_, err := conn.Write([]byte("bonk zoops"))
	if err != nil {
		t.Fatalf("err not nil: %v\n", err)
	}
	conn.Close()
	conn = wsConnection("localhost:8333", "/donk", t)
	_, err = conn.Write([]byte("donk zoops"))
	if err != nil {
		t.Fatalf("err not nil: %v\n", err)
	}
	listener.wg.Wait()
}

// TestNewMultiServer demonstrates the ability to register
// the same listener to multiple services.
func TestNewMultiServer(t *testing.T) {
	serve := NewGenServe()
	listener := NewListener()
	addrTcp := &GenAddr{Proto: "tcp", Addr: ":8334", Path: ""}
	addrUdp := &GenAddr{Proto: "udp", Addr: ":8335", Path: ""}
	addrWs := &GenAddr{Proto: "ws", Addr: ":8336", Path: "/bonk"}
	listener.wg.Add(4)
	serve.ListenTCP(addrTcp, listener)
	serve.ListenUDP(addrUdp, listener)
	serve.MapWSPath(addrWs, listener)
	serve.ListenWS(addrWs.Addr)
	// tcp
	conn := tcpConnection("localhost:8334", t)
	conn.Write([]byte("TCP zoops"))
	conn.Close()

	// udp
	conn = udpConnection("localhost:8335", t)
	conn.Write([]byte("UDP zoops"))

	// ws
	conn = wsConnection("localhost:8336", "/bonk", t)
	_, err := conn.Write([]byte("WS zoops"))
	if err != nil {
		t.Fatalf("err not nil: %v\n", err)
	}
	listener.wg.Wait()
}

func udpConnection(addr string, t *testing.T) net.Conn {
	return connect("udp", addr, t)
}

func tcpConnection(addr string, t *testing.T) net.Conn {
	return connect("tcp", addr, t)
}

func wsConnection(addr, path string, t *testing.T) *websocket.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing", err)
	}
	client, errWS := websocket.NewClient(newConfig(t, addr, path), conn)
	if errWS != nil {
		t.Fatalf("WebSocket handshake error: %v\n", errWS)
	}
	return client
}

func connect(proto, addr string, t *testing.T) net.Conn {
	conn, err := net.Dial(proto, addr)
	if err != nil {
		t.Fatalf("error connecting to %s due to: %v\n", addr, err)
	}
	return conn
}

func newConfig(t *testing.T, addr, path string) *websocket.Config {
	config, _ := websocket.NewConfig(fmt.Sprintf("ws://%s%s", addr, path), "http://localhost")
	return config
}
