package main

import (
	"encoding/json"
	"fmt"
	"gobwas-quizy/pool"
	"gobwas-quizy/role"

	"log"
	"net/http"
	"syscall"

	"github.com/gorilla/websocket"
)

type payload struct {
	Status  int         `json:"status" bson:"status"`
	Message interface{} `json:"message" bson:"message"`
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

	http.HandleFunc("/ws/join", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)

		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		nickname := r.URL.Query().Get("nickname")
		room := pool.Search(code)
		player := role.NewEntity(nickname, role.MEMBER_ROLE, conn)
		room.Add(player)

	})

	http.HandleFunc("/ws/play", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
		if err != nil {
			http.Error(w, "could not open websocket connection", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		admin := role.NewEntity(code, role.ADMIN_ROLE, conn)
		room, err := pool.NewRoom()
		room.Add(admin)
		go room.Start()
		pool.Add(code, room)
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		nickname := r.URL.Query().Get("nickname")

		if r.Method != http.MethodGet {
			http.Error(w, "Just Allow GET Method", http.StatusBadRequest)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Content-Type", "application/json")
		val, room := pool.Check(code, nickname)
		if !val {
			pld := payload{
				Status:  0,
				Message: "Room not Found",
			}

			bit, _ := json.Marshal(pld)
			w.Write(bit)
			return
		}

		if room != nil {
			pld := payload{
				Status:  0,
				Message: "Your Alredy on Room",
			}

			bit, _ := json.Marshal(pld)
			w.Write(bit)
			return
		}

		pld := payload{
			Status:  1,
			Message: "Room is Ready",
		}

		bit, _ := json.Marshal(pld)
		w.Write(bit)

	})

	fmt.Println("server run :8083")
	if err := http.ListenAndServe(":8083", nil); err != nil {
		log.Fatal(err)
	}
}
