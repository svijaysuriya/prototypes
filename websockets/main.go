package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	// go func() {
	// 	ticker := time.NewTicker(10 * time.Second)
	// 	for t := range ticker.C {
	// 		p := []byte("time: " + t.String())
	// 		println("ping: " + string(p))
	// 		if err := conn.WriteMessage(websocket.TextMessage, p); err != nil {
	// 			return
	// 		}
	// 	}
	// }()
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		println("received: " + string(p))
		p = []byte("received: " + string(p))
		if err := conn.WriteMessage(messageType, p); err != nil {
			return
		}
	}
}
func main() {
	http.HandleFunc("/ws", websocketHandler)
	http.Handle("/", http.FileServer(http.Dir("./")))
	http.ListenAndServe(":8080", nil)
}
