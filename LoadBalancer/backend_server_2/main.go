package main

import (
	"fmt"
	"net/http"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr, "backend server 2 says: Keep Implementing")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("server 2: Keep Implementing\n"))
}
func main() {
	http.HandleFunc("/", sayHello)
	fmt.Println("started server 2 in port 9000")
	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		panic(err)
	}

}
