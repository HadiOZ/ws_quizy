package roles

import "github.com/gorilla/websocket"

type Entity interface {
	Conn() *websocket.Conn
	Code() string
	RWMessage(map[int]Entity) error
}
