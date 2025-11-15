package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Backend struct {
	Host  string
	Port  string
	Total int
}

type LB struct {
	Servers  []Backend
	LastUsed int
}

var mu sync.Mutex

func NewLB() *LB {
	return &LB{
		Servers: []Backend{
			{"localhost", "8000", 0},
			{"localhost", "9000", 0},
		},
		LastUsed: 1,
	}
}

func (lb *LB) Proxy(conn net.Conn, reqId string) {

	lb.LastUsed = (lb.LastUsed + 1) % len(lb.Servers)
	var buggy []byte
	conn.Read(buggy)
	fmt.Println(conn.RemoteAddr().String(), "\tserver going to be used is ", lb.LastUsed, " request id = ", reqId, "\n", buggy)

	mu.Lock()
	lb.Servers[lb.LastUsed].Total++
	backend := lb.Servers[lb.LastUsed]
	mu.Unlock()

	backendConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", backend.Host, backend.Port))
	if err != nil {
		conn.Write([]byte("HTTP/1.1 500 InternalServerError\r\n\r\nbackend server not avialable"))
		log.Fatal("err while dailing on host:port ", fmt.Sprintf("%s:%s", backend.Host, backend.Port), " => ", err)
	}
	// fmt.Println(reqId, "\t", "going to copy the content from conn to backendConn")
	go func() {
		_, err := io.Copy(backendConn, conn)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 500 InternalServerError\r\n\r\nbackend server not avialable"))
			log.Fatal("err while coping ", err)

		}
		// fmt.Println(reqId, "\t", copiedBytes, "going to copy the content from backendConn to conn")

	}()
	go func() {
		_, err := io.Copy(conn, backendConn)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 500 InternalServerError\r\n\r\nbackend server not avialable"))
			log.Fatal("err while coping ", err)

		}
		// fmt.Printsln(reqId, "\t", copiedBytes, "conn: successfully copied from backendConn")
	}()
	// conn.Close()
}

func main() {
	lb := NewLB()
	listener, err := net.Listen("tcp", ":7878")
	if err != nil {
		log.Fatal("err while listening on port 7878 => ", err)
	}
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal("err while accepting connectiong on port 7878 => ", err)
		}
		go lb.Proxy(connection, time.Now().String())
	}
}
