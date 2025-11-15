package main

import (
	"fmt"
	"sync"
	"time"
)

var Q []interface{}

type BlockingQ struct {
	ch   chan bool
	Q    []obj
	mu   sync.Mutex
	size int
}

type obj struct {
}

func (bq BlockingQ) Get() obj {
	<-bq.ch

	bq.mu.Lock()
	ret := bq.Q[0]
	bq.Q = bq.Q[1:]
	bq.mu.Unlock()

	return ret
}

func (bq BlockingQ) Put(ob obj) {
	bq.mu.Lock()
	bq.Q = append(bq.Q, ob)
	bq.mu.Unlock()

	bq.ch <- true
}

func newBlockingQ(size int) BlockingQ {
	mu := sync.Mutex{}
	blockQ := BlockingQ{
		ch: make(chan bool, size),
		Q:  make([]obj, 0),
		mu: mu,
	}
	for i := 0; i < size; i++ {
		blockQ.Q = append(blockQ.Q, obj{})
		blockQ.ch <- true
	}
	return blockQ
}
func main() {
	poolSize := 10
	start := time.Now()
	// Q = make([]interface{}, poolSize)
	blockQ := newBlockingQ(poolSize)

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(user int) {
			defer wg.Done()
			ob := blockQ.Get()
			fmt.Println("got conn obj for user = ", user)
			time.Sleep(1 * time.Second)
			blockQ.Put(ob)
			fmt.Println("put user = ", user)

		}(i)
	}
	wg.Wait()
	fmt.Println("time taken = ", time.Since(start).String())
}
