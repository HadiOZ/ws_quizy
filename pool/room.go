package pool

import (
	roles "gobwas-quizy/roles"
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
	connections map[int]roles.Entity
	lock        *sync.RWMutex
}

func NewRoom() (*Room, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &Room{
		fd:          fd,
		connections: make(map[int]roles.Entity),
		lock:        &sync.RWMutex{},
	}, nil

}

func (r *Room) LenConn() int {
	return len(r.connections)
}

func (r *Room) Add(entity roles.Entity) error {
	fd := WebsocketFD(entity.Conn())
	err := unix.EpollCtl(r.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.connections) == 0 {
		r.Admin = fd
	}
	r.connections[fd] = entity
	if len(r.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(r.connections))
	}
	return nil
}

func (r *Room) Remove(entity roles.Entity) error {
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

func (r *Room) Wait() ([]roles.Entity, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(r.fd, events, 100)
	if err != nil {
		return nil, err
	}
	r.lock.RLock()
	defer r.lock.RUnlock()
	var connections []roles.Entity
	for i := 0; i < n; i++ {
		conn := r.connections[int(events[i].Fd)]
		connections = append(connections, conn)
	}
	return connections, nil
}

func (r *Room) Start() {
	for {
		connections, err := r.Wait()
		if len(connections) == 0 {
			break
		}
		if err != nil {
			log.Printf("Failed to epoll wait %v", err)
			continue
		}
		for _, conn := range connections {
			if err := conn.RWMessage(r.connections); err != nil {
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
