package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"net"
	"testing"
	"time"
)

type DefaultObserve struct {
}

func (d *DefaultObserve) OnConnect(conn *GenConn) {
	fmt.Printf("Yup got connection %v\n", conn)
}

func (d *DefaultObserve) OnDisconnect(conn *GenConn) {
	fmt.Printf("Yup connection dropped %v\n", conn)
}

func (d *DefaultObserve) OnRecv(conn *GenConn, data []byte) {
	fmt.Printf("Yup got data %s from connection %v\n", conn, string(data))
}

func (d *DefaultObserve) OnError(conn *GenConn) {
	fmt.Printf("Yup got error %v\n", conn)
}

func TestNewTcpServer(t *testing.T) {
	addr := ":8333"
	serve := NewGenServe()
	conn := make(chan *GenConn, 1)
	serve.ListenTCP(addr, conn)
	for {
		select {
		case newConn := <-conn:
			fmt.Printf("yup listening... and got connection: %v\n", newConn)
			return
		default:
			tcpConnection("localhost"+addr, t)
		}
	}
}

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

func udpConnection(addr string, t *testing.T) {
	connect("udp", addr, t)
}

func tcpConnection(addr string, t *testing.T) {
	connect("tcp", addr, t)
}

func wsConnection(addr, path string, t *testing.T) {
	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal("dialing", err)
	}
	_, err = websocket.NewClient(newConfig(t, addr, path), client)
	if err != nil {
		t.Errorf("WebSocket handshake error: %v", err)
		return
	}
}

func connect(proto, addr string, t *testing.T) {
	_, err := net.Dial(proto, addr)
	if err != nil {
		t.Fatalf("error connecting to %s due to: %v", addr, err)
	}
}

func newConfig(t *testing.T, addr, path string) *websocket.Config {
	config, _ := websocket.NewConfig(fmt.Sprintf("ws://%s%s", addr, path), "http://localhost")
	return config
}
