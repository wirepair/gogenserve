package gogenserve

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"net"
)

type GenServer interface {
	Listen(serveType string) error
	Dispatch(msg []byte) error
}

type GenServe struct {
}

func NewGenServe() *GenServe {
	return &GenServe{}
}

func (g *GenServe) Listen(props *ListenProps) error {
	switch props.Type {
	case "socket":
		g.socketListen(props.Proto, props.Addr)
	case "websocket":
		g.webSocketListen(addr)
	}
}

func (g *GenServe) socketListen(proto, addr string) error {
	return nil
}

func (g *GenServe) webSocketListen(addr string) error {
	return nil
}
