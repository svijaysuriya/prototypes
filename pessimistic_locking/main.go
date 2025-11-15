package main

import (
	"fmt"
	"runtime"
	"sync"
)

var tickets = 1000
var wg sync.WaitGroup // to synchronize the go routines
var mutex sync.Mutex  // to pessimistically lock the critical section ()

func reduceTicketCount() {
	mutex.Lock()
	tickets--
	mutex.Unlock()
	wg.Done()
}
func bookTicket() {
	// assuming concurrent 1000 users
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go reduceTicketCount()
		if i%23 == 0 {
			fmt.Println("rand matched #goroutines = ", runtime.NumGoroutine())
		}
	}
}
func main() {
	// start := time.Now()
	// bookTicket()
	// fmt.Println("#goroutines = ", runtime.NumGoroutine())
	// wg.Wait()
	// fmt.Println("#goroutines = ", runtime.NumGoroutine())
	// end := time.Since(start).String()
	// fmt.Println("tickets = ", tickets, "\ntime taken for concurrent execution = ", end)
	// fmt.Println("#goroutines = ", runtime.NumGoroutine())

	// fmt.Println("#cores = ", runtime.NumCPU(), "\nGOMAXPROCS = ", runtime.GOMAXPROCS(-2))

	fmt.Println("=--= OBD Issue =--=")
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			enrichedData := getEnrichedDataFromGovtOrIntegration("123")
			fmt.Println(enrichedData)
			wg.Done()
		}()
	}
	wg.Wait()
}
