package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"syscall"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var epoller *epoll
var erooms *eroom

type payload struct {
	From    string
	Type    string
	Message interface{}
}

func main() {
	// Increase resources limitations
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		panic(err)
	}

	// Enable pprof hooks
	// go func() {
	// 	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
	// 		log.Fatalf("pprof failed: %v", err)
	// 	}
	// }()

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

	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		content, err := ioutil.ReadFile("play.html")
		if err != nil {
			http.Error(w, "Could not open requested file", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", content)
	})

	http.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		content, err := ioutil.ReadFile("join.html")
		if err != nil {
			http.Error(w, "Could not open requested file", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", content)
	})

	http.HandleFunc("/ws/join", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)

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
		conn, _, _, err := ws.UpgradeHTTP(r, w)

		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		n := mkAdmin(conn, code)
		epoller.Add(n)
	})

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
			if msg, _, err := wsutil.ReadClientData(conn.Conn); err != nil {
				if err := epoller.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Close()
			} else {
				for _, c := range conn.players {
					wsutil.WriteClientMessage(c, ws.OpText, msg)
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
			if msg, _, err := wsutil.ReadClientData(conn.Conn); err != nil {
				if err := erooms.Remove(conn); err != nil {
					log.Printf("Failed to remove %v", err)
				}
				conn.Close()
			} else {
				wsutil.WriteClientMessage(*conn.admin, ws.OpText, msg)
			}
		}
	}
}
