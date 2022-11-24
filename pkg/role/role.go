package role

import (
	"github.com/gorilla/websocket"
)

const (
	ADMIN_ROLE  = 0
	MEMBER_ROLE = 1
)

type Entity struct {
	Role     int
	conn     *websocket.Conn
	uniqcode string
}

func (e *Entity) Conn() *websocket.Conn {
	return e.conn
}

func (e *Entity) Uniqcode() string {
	return e.uniqcode
}

func NewEntity(uniqcode string, role int, connection *websocket.Conn) Entity {
	return Entity{
		Role:     role,
		conn:     connection,
		uniqcode: uniqcode,
	}
}

func (e *Entity) RWMessage(handel func(uniqcode string, sender *websocket.Conn) error) error {
	err := handel(e.uniqcode, e.conn)
	return err
}

// need
// func (p *Entity) RWMessage() error {
// 	t, m, err := p.conn.ReadMessage()
// 	if err != nil {
// 		return fmt.Errorf("can't read message from %s", p.code)
// 	}

// 	err = target[p.admin].Conn().WriteMessage(t, m)
// 	if err != nil {
// 		return fmt.Errorf("can't write message to %s", target[p.admin].Code())
// 	}

// 	return nil
// }
