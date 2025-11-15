package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func manageConnection(conn net.Conn) {
	buffer := make([]byte, 2000)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Fatal("R error while reading input from user", err)
	}
	//     HTTP/1.1 200 OK

	fmt.Println("processing request = ", string(buffer))
	time.Sleep(10 * time.Second)
	writeString := "HTTP/1.1 200 OK\r\n\r\nhello world\r\n"
	_, err = conn.Write([]byte(writeString))
	if err != nil {
		log.Fatal("W error while write to user", err)
	}
	err = conn.Close()
	if err != nil {
		log.Fatal("Close Error = ", err)
	}
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("L error while creating a TCP listener", err)
	}
	for {
		fmt.Println("waiting for tcp connection")
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal("A error while accepting a TCP connection", err)
		}
		fmt.Println("connected with client = ", connection.LocalAddr())
		go manageConnection(connection)
	}
}
