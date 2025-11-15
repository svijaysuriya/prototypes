package main

import (
	"fmt"
	"net/http"
	"time"
)

func ServerSentEventsHandler(w http.ResponseWriter, r *http.Request) {
	ticker := time.NewTicker(2 * time.Second)
	// w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Add("Content-Type", "text/event-stream")

	for t := range ticker.C {
		fmt.Println(r.Header.Get("sse"), "\t", t.String())
		fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf("time %s", t.String()))
		w.(http.Flusher).Flush()

		w.Write([]byte("data: " + "vjs"))
	}
}

func main() {
	http.HandleFunc("/sse", ServerSentEventsHandler)
	http.Handle("/", http.FileServer(http.Dir("./asset")))

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		panic(err)
	}
}
