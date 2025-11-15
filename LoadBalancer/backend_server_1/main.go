package main

import (
	"fmt"
	"net/http"
)

// func handleConnection(conn net.Conn) {
// 	fmt.Println(conn.RemoteAddr().String(), " backend server 1 connected")
// 	conn.Write([]byte("Hello from backend server 1\n"))
// 	conn.Close()
// }

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr, "new http request received on backend server 1")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from backend server 1\n"))
}
func main() {
	http.HandleFunc("/", sayHello)
	fmt.Println("8000 port server 1 started")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		panic(err)
	}
}
