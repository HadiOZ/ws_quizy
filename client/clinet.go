package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var url string

func init() {
	flag.StringVar(&url, "url", "", "required url to connect websocket server")
	flag.Parse()
}

func Connect(url string) (*websocket.Conn, *http.Response, error) {
	head := http.Header{}
	head.Set("User-Agent", "ozmonday")
	return websocket.DefaultDialer.Dial(url, head)
}

func ScaneInput(ctx context.Context, conn *websocket.Conn) {
	for {
		var msg string
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Scanln(&msg)
		}
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

func ReadMessage(ctx context.Context, conn *websocket.Conn, msg chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			msg <- string(data)
		}
	}
}

func main() {
	msg := make(chan string)
	conn, _, err := Connect(url)
	if err != nil {
		log.Println(err.Error())
		return
	}

	go ScaneInput(context.Background(), conn)
	go ReadMessage(context.Background(), conn, msg)

	for m := range msg {
		fmt.Println(m)
	}
}
