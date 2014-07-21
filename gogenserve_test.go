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
	if size != 0 {
		d.wg.Done()
	}
}

func (d *DefaultListener) OnError(conn *GenConn, err error) {
	fmt.Printf("Yup got error on connection %v: %v\n", conn, err)
}

func (d *DefaultListener) Addr() *GenAddr {
	return d.addr
}

func NewListener(proto, addr string) *DefaultListener {
	l := new(DefaultListener)
	l.addr = &GenAddr{Proto: proto, Addr: addr}
	l.wg = new(sync.WaitGroup)
	l.wg.Add(1) // for connection
	return l
}

func TestNewTcpServer(t *testing.T) {
	listener := NewListener("tcp", ":8333")
	listener.wg.Add(1)
	serve := NewGenServe()
	fmt.Printf("listening...")
	serve.ListenTCP(listener)
	conn := tcpConnection("localhost:8333", t)
	fmt.Printf("Sending zoops")
	conn.Write([]byte("zooops"))
	conn.Close()
	listener.wg.Wait()
}

/*
func TestNewWSServer(t *testing.T) {
	serve := NewGenServe()
	conn := make(chan *GenConn, 1)
	serve.MapWSPath("/bonk", conn)
	serve.ListenWS(":8333")
	for {
		select {
		case newConn := <-conn:
			fmt.Printf("yup listening... and got connection: %v\n", newConn)
			return
		default:
			wsConnection("localhost:8333", "/bonk", t)
		}
	}
}

func TestNewUDPServer(t *testing.T) {
	addr := "127.0.0.1:8333"
	serve := NewGenServe()
	conn := make(chan *GenConn, 1)
	serve.ListenUDP(addr, conn)
	for {
		select {
		case newConn := <-conn:
			fmt.Printf("yup listening... and got connection: %v\n", newConn)
			return
		default:
			udpConnection("localhost"+addr, t)
		}
	}
}

func TestNewMultiServer(t *testing.T) {
	serve := NewGenServe()
	tcpConn := make(chan *GenConn, 1)
	wsConn := make(chan *GenConn, 1)
	serve.MapWSPath("/bonk", wsConn)
	serve.ListenWS(":8333")
	serve.ListenTCP(":8444", tcpConn)
	connCount := 0
	for {
		select {
		case newConn := <-wsConn:
			fmt.Printf("got ws connection: %v\n", newConn)
			connCount++
			if connCount == 2 {
				return
			}
		case newConn := <-tcpConn:
			fmt.Printf("got tcp connection: %v\n", newConn)
			connCount++
			if connCount == 2 {
				return
			}
		case <-time.After(time.Second * 10):
			fmt.Printf("timed out.\n")
			return
		default:
			tcpConnection("localhost:8444", t)
			wsConnection("localhost:8333", "/bonk", t)
		}
	}
}
*/
func udpConnection(addr string, t *testing.T) net.Conn {
	return connect("udp", addr, t)
}

func tcpConnection(addr string, t *testing.T) net.Conn {
	return connect("tcp", addr, t)
}

func wsConnection(addr, path string, t *testing.T) net.Conn {
	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing", err)
	}
	_, err = websocket.NewClient(newConfig(t, addr, path), client)
	if err != nil {
		t.Fatalf("WebSocket handshake error: %v", err)
	}
	return client
}

func connect(proto, addr string, t *testing.T) net.Conn {
	conn, err := net.Dial(proto, addr)
	if err != nil {
		t.Fatalf("error connecting to %s due to: %v", addr, err)
	}
	return conn
}

func newConfig(t *testing.T, addr, path string) *websocket.Config {
	config, _ := websocket.NewConfig(fmt.Sprintf("ws://%s%s", addr, path), "http://localhost")
	return config
}
