package roles

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type Admin struct {
	conn *websocket.Conn
	code string
}

func (a *Admin) Conn() *websocket.Conn {
	return a.conn
}

func (a *Admin) Code() string {
	return a.code
}

func (a *Admin) RWMessage(target map[int]Entity) error {
	t, m, err := a.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("can't read message from %s", a.code)
	}

	for _, conn := range target {
		if conn.Code() != a.Code() {
			err := conn.Conn().WriteMessage(t, m)
			if err != nil {
				return fmt.Errorf("can't write message to %s", conn.Code())
			}
		}
	}
	return nil
}

func NewAdmin(code string, connection *websocket.Conn) Admin {
	return Admin{
		conn: connection,
		code: code,
	}
}
