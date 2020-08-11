package main

import (
	"errors"
	"log"
	"reflect"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	"golang.org/x/sys/unix"
)

type admin struct {
	*websocket.Conn
	code    string
	players map[string]*websocket.Conn
	lock    *sync.RWMutex
}

func mkAdmin(conn *websocket.Conn, key string) admin {
	return admin{
		Conn:    conn,
		code:    key,
		players: make(map[string]*websocket.Conn),
		lock:    &sync.RWMutex{},
	}
}

func (a *admin) add(conn *websocket.Conn, key string) {
	log.Println("can run add 1")
	a.players[key] = conn
	log.Println("can run add 2")
	log.Println("can run add 3")
}

type epoll struct {
	fd          int
	connections map[int]admin
	lock        *sync.RWMutex
}

func mkEpoll() (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	return &epoll{
		fd:          fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]admin),
	}, nil
}

func (e *epoll) addPlayer(code string, conn *websocket.Conn, key string) (*websocket.Conn, error) {
	for _, a := range e.connections {
		if a.code == code {
			a.lock.Lock()
			defer a.lock.Unlock()
			a.players[key] = conn
			return a.Conn, nil
		}
	}
	err := errors.New("admin not found")
	return nil, err
}

func (e *epoll) checkAdmin(code string, key string) (bool, error) {
	for _, a := range e.connections {
		if a.code == code {
			a.lock.Lock()
			defer a.lock.Unlock()
			for keys, _ := range a.players {
				if keys == key {
					return true, nil
				}
			}
			return false, nil
		}
	}
	err := errors.New("admin not found")
	return false, err
}

func (e *epoll) Add(conn admin) error {
	// Extract file descriptor associated with the connection
	fd := websocketFD(conn.Conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.connections[fd] = conn
	if len(e.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.connections))
	}
	return nil
}

func (e *epoll) Remove(conn admin) error {
	fd := websocketFD(conn.Conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.connections, fd)
	if len(e.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.connections))
	}
	return nil
}

func (e *epoll) Wait() ([]admin, error) {
	events := make([]unix.EpollEvent, 100)
	n, err := unix.EpollWait(e.fd, events, 100)
	if err != nil {
		return nil, err
	}
	e.lock.RLock()
	defer e.lock.RUnlock()
	var connections []admin
	for i := 0; i < n; i++ {
		conn := e.connections[int(events[i].Fd)]
		connections = append(connections, conn)
	}
	return connections, nil
}

func websocketFD(conn *websocket.Conn) int {
	connVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn").Elem()
	tcpConn := reflect.Indirect(connVal).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}
