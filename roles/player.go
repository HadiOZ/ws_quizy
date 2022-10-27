package roles

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type Player struct {
	admin int
	conn  *websocket.Conn
	code  string
}

func (a *Player) Conn() *websocket.Conn {
	return a.conn
}

func (a *Player) Code() string {
	return a.code
}

func NewPlayer(code string, admin int, connection *websocket.Conn) Player {
	return Player{
		admin: admin,
		conn:  connection,
		code:  code,
	}
}

// need
func (p *Player) RWMessage(target map[int]Entity) error {
	t, m, err := p.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("can't read message from %s", p.code)
	}

	err = target[p.admin].Conn().WriteMessage(t, m)
	if err != nil {
		return fmt.Errorf("can't write message to %s", target[p.admin].Code())
	}

	return nil
}
