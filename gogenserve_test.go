package gogenserve

import (
	"fmt"
	"testing"
	"time"
)

func TestNewTcpServer(t *testing.T) {
	serve := NewGenServe()
	conn := make(chan *GenConn, 1)
	serve.ListenTCP(":8333", conn)

	select {
	case newConn := <-conn:
		fmt.Printf("yup listening... and got connection: %v\n", newConn)
	}

}

func TestNewWSServer(t *testing.T) {
	serve := NewGenServe()
	conn := make(chan *GenConn, 1)

	serve.ListenWS(":8333", "/bonk", conn)
	select {
	case newConn := <-conn:
		fmt.Printf("yup listening... and got connection: %v\n", newConn)
	}
}

func TestMultiServer(t *testing.T) {
	serve := NewGenServe()
	tcpConn := make(chan *GenConn, 1)
	wsConn := make(chan *GenConn, 1)

	serve.ListenWS(":8333", "/bonk", wsConn)
	serve.ListenTCP(":8444", tcpConn)
	for {
		select {
		case newConn := <-wsConn:
			fmt.Printf("got connection: %v\n", newConn)
		case newConn := <-tcpConn:
			fmt.Printf("got tcp connection: %v\n", newConn)
		case <-time.After(time.Second * 25):
			fmt.Printf("timed out.\n")
			return
			//default:
			//		fmt.Printf("nothing ready.\n")
		}
	}

}
