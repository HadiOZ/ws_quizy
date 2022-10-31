package pool

import (
	"fmt"
	"gobwas-quizy/role"
	"log"
	"reflect"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	"golang.org/x/sys/unix"
)

type Room struct {
	fd          int
	Admin       []int
	connections map[int]role.Entity
	lock        *sync.RWMutex
}

func NewRoom() (*Room, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &Room{
		fd:          fd,
		connections: make(map[int]role.Entity),
		lock:        &sync.RWMutex{},
	}, nil

}

func (r *Room) Len() int {
	return len(r.connections)
}

func (r *Room) Add(entity role.Entity) error {
	fd := WebsocketFD(entity.Conn())
	err := unix.EpollCtl(r.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	if entity.Role == role.ADMIN_ROLE {
		r.Admin = append(r.Admin, fd)
	}
	r.connections[fd] = entity
	if len(r.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(r.connections))
	}
	return nil
}

func (r *Room) Remove(entity role.Entity) error {
	fd := WebsocketFD(entity.Conn())
	err := unix.EpollCtl(r.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.connections, fd)
	if len(r.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(r.connections))
	}
	return nil
}

func (r *Room) Wait() ([]role.Entity, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(r.fd, events, 100)
	if err != nil {
		return nil, err
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	var connections []role.Entity
	for i := 0; i < n; i++ {
		conn := r.connections[int(events[i].Fd)]
		connections = append(connections, conn)
	}
	return connections, nil
}

func (r *Room) broadcash(uniqcode string, sender *websocket.Conn) error {
	t, m, err := sender.ReadMessage()
	log.Printf("message from %s : %s\n", uniqcode, string(m))
	if err != nil {
		return fmt.Errorf("can't read message from %s", uniqcode)
	}

	for _, val := range r.connections {
		if val.Uniqcode() == uniqcode {
			continue
		}
		err := val.Conn().WriteMessage(t, m)
		if err != nil {
			return fmt.Errorf("can't write message to %s", val.Uniqcode())
		}
	}
	return nil
}

func (r *Room) peerToPeer(uniqcode string, sender *websocket.Conn) error {
	t, m, err := sender.ReadMessage()
	if err != nil {
		return fmt.Errorf("can't read message from %s", uniqcode)
	}

	for _, admin := range r.Admin {
		adm := r.connections[admin]
		err := adm.Conn().WriteMessage(t, m)
		if err != nil {
			return fmt.Errorf("can't write message to %s", adm.Uniqcode())
		}
	}

	return nil
}

func (r *Room) Start() {
	for {
		connections, err := r.Wait()

		if err != nil {
			log.Printf("Failed to epoll wait %v", err)
			continue
		}
		for _, conn := range connections {
			switch conn.Role {
			case role.ADMIN_ROLE:
				err = conn.RWMessage(r.broadcash)
			case role.MEMBER_ROLE:
				err = conn.RWMessage(r.peerToPeer)
			}

			if err != nil {
				if err := r.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Conn().Close()
			}

		}
	}
}

func WebsocketFD(conn *websocket.Conn) int {
	connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(connVal).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}
