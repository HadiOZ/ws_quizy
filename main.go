package main

import (
	"fmt"
	"log"
	"net/http"
	"syscall"

	"github.com/gorilla/websocket"
)

var epoller *epoll
var erooms *eroom

type payload struct {
	From    string
	Type    string
	Message interface{}
}

func main() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	// Start epoll
	var err error
	epoller, err = mkEpoll()
	if err != nil {
		panic(err)
	}

	erooms, err = mkEroom()
	if err != nil {
		panic(err)
	}

	go startepoll()
	go starteroom()

	http.HandleFunc("/ws/join", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		nickname := r.URL.Query().Get("nickname")
		adm, err := epoller.addPlayer(code, conn, nickname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		n := player{}
		n.admin = adm
		n.name = nickname
		n.Conn = conn
		if err := erooms.Add(n); err != nil {
			log.Printf("Failed to add connection %v", err)
			n.Conn.Close()
			conn.Close()
		}
	})

	http.HandleFunc("/ws/play", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		n := mkAdmin(conn, code)
		epoller.Add(n)
	})
	fmt.Println("server run")
	if err := http.ListenAndServe(":8083", nil); err != nil {
		log.Fatal(err)
	}
}

func startepoll() {
	for {
		connections, err := epoller.Wait()
		if err != nil {
			log.Printf("Failed to epoll wait %v", err)
			continue
		}
		for _, conn := range connections {
			if conn.Conn == nil {
				break
			}
			if t, m, err := conn.Conn.ReadMessage(); err != nil {
				if err := epoller.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Close()
			} else {
				for _, c := range conn.players {
					if err := c.WriteMessage(t, m); err != nil {
						log.Printf("Failed to send  message %v", err)
					}
				}
			}
		}
	}
}

func starteroom() {
	for {
		connections, err := erooms.Wait()
		if err != nil {
			log.Printf("Failed to epoll wait %v", err)
			continue
		}
		for _, conn := range connections {
			if conn.Conn == nil {
				break
			}
			if t, m, err := conn.ReadMessage(); err != nil {
				if err := erooms.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Close()
			} else {
				if err := conn.admin.WriteMessage(t, m); err != nil {
					log.Printf("Failed to send  message %v", err)
				}
			}
		}
	}
}
