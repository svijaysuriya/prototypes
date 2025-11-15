// package main

// import (
// 	"fmt"
// 	"log"
// 	"net"
// 	"syscall"
// )

// func main() {
// 	listener, err := net.Listen("tcp", "localhost:7379")
// 	if err != nil {
// 		log.Fatal("error while listening on port 7379", err)
// 	}
// 	log.Println("listening on port 7379")
// 	numberOfConnections := 0
// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Fatal("error while accepting connection", err)
// 		}
// 		numberOfConnections++
// 		log.Println("new connection accepted from client => ", conn.RemoteAddr().String(), "\t numberOfConnections = ", numberOfConnections)
// 		syscall.Kqueue()
// 		for {
// 			buff := make([]byte, 1024)
// 			_, err := conn.Read(buff)
// 			if err != nil {
// 				log.Println("error while reading from client", err)
// 				numberOfConnections--
// 				break
// 			}

// 			fmt.Println("from client => ", conn.RemoteAddr().String(), "\tcommand:", string(buff))

// 			_, err = conn.Write(buff)
// 			if err != nil {
// 				log.Println("error while writing to client", err)
// 				numberOfConnections--
// 				break
// 			}
// 			fmt.Println(" response to client => ", conn.RemoteAddr().String())
// 		}

// 	}
// }

//go:build darwin || freebsd || netbsd || openbsd
// +build darwin freebsd netbsd openbsd

package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	// ...
	defer ln.Close()

	// Get the listener's file, not just the FD
	listenerFile, err := getSocketFile(ln)
	if err != nil {
		fmt.Println("Error getting socket file:", err)
		os.Exit(1)
	}
	// Defer closing the *file*
	defer listenerFile.Close()

	// Get the FD from the file
	listenerFd := int(listenerFile.Fd())
	fmt.Printf("Listening on :8080 (FD: %d)\n", listenerFd)

	// 2. Create a kqueue instance
	kq, err := syscall.Kqueue()
	if err != nil {
		fmt.Println("Error creating kqueue:", err)
		os.Exit(1)
	}
	defer syscall.Close(kq)

	// 3. Register the listener FD with kqueue
	// We create a "change" event to submit to kqueue
	change := syscall.Kevent_t{
		Ident:  uint64(listenerFd),                 // Identifier (file descriptor)
		Filter: syscall.EVFILT_READ,                // Watch for "read" events
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE, // Add this event & enable it
	}

	// Register the change with Kevent. We don't need to wait for events yet.
	// changelist, eventlist, timeout
	_, err = syscall.Kevent(kq, []syscall.Kevent_t{change}, nil, nil)
	if err != nil {
		fmt.Println("Error registering listener with kqueue:", err, err.Error())
		os.Exit(1)
	}

	// 4. Start the event loop
	events := make([]syscall.Kevent_t, 128) // Buffer for retrieved events
	for {
		// Wait for events. Timeout is nil, so it blocks indefinitely.
		// We pass nil for changelist because we're only polling, not registering.
		nevents, err := syscall.Kevent(kq, nil, events, nil)
		if err != nil {
			// EINTR is an "interrupted" syscall, often fine to just continue
			if err == syscall.EINTR {
				continue
			}
			fmt.Println("Error in kqueue wait:", err)
			break
		}

		// Handle all ready events
		for i := 0; i < nevents; i++ {
			ev := events[i]
			fd := int(ev.Ident)

			// Handle client disconnection
			if ev.Flags&syscall.EV_EOF != 0 {
				fmt.Println("Client disconnected (FD:", fd, ")")
				// Closing the FD automatically removes it from kqueue
				syscall.Close(fd)
				continue
			}

			if fd == listenerFd {
				// --- Event is on the listener: New connection ---
				conn, _, err := syscall.Accept(listenerFd)
				if err != nil {
					fmt.Println("Error accepting connection:", err)
					continue
				}
				// Set new connection to non-blocking
				syscall.SetNonblock(conn, true)

				// Register the new connection with kqueue
				connChange := syscall.Kevent_t{
					Ident:  uint64(conn),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR, // EV_CLEAR = edge-triggered
				}
				// Submit this new change
				if _, err := syscall.Kevent(kq, []syscall.Kevent_t{connChange}, nil, nil); err != nil {
					fmt.Println("Error adding conn to kqueue:", err)
					syscall.Close(conn)
				}
				fmt.Println("Accepted connection (FD:", conn, ")")

			} else {
				// --- Event is on a client connection: Data ready ---
				buf := make([]byte, 1024)
				n, err := syscall.Read(fd, buf)
				if err != nil || n == 0 {
					// Error or client disconnected (should be caught by EV_EOF, but good to check)
					if err != nil {
						fmt.Println("Error reading from conn:", err)
					}
					syscall.Close(fd)
					fmt.Println("Closed connection (FD:", fd, ")")
					continue
				}

				// Echo the data back
				fmt.Printf("Received %d bytes from FD %d: %s", n, fd, string(buf[:n]))
				_, err = syscall.Write(fd, buf[:n])
				if err != nil {
					fmt.Println("Error writing to conn:", err)
				}
			}
		}
		fmt.Println("\n------ new poll for events -------\n")
	}
}

// Helper function to get the raw file
func getSocketFile(ln net.Listener) (*os.File, error) {
	tcpListener, ok := ln.(*net.TCPListener)
	if !ok {
		return nil, fmt.Errorf("listener was not a *net.TCPListener")
	}

	file, err := tcpListener.File()
	if err != nil {
		return nil, err
	}

	// DO NOT close the file here
	return file, nil
}
