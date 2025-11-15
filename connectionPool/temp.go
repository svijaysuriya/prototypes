package main

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

func newConn() *sql.DB {
	db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/cricket_db")
	if err != nil {
		panic(err)
	}
	return db
}

type ConnectionPool struct {
	maxConn int
	ch      chan *sql.DB
}

func NewConnectionPool(maxCon int) *ConnectionPool {
	cp := ConnectionPool{
		maxConn: maxCon,
		ch:      make(chan *sql.DB, maxCon),
	}
	for i := 0; i < maxCon; i++ {
		cp.ch <- newConn()
	}
	return &cp
}

func Get(cp *ConnectionPool) *sql.DB {
	return <-cp.ch
}

func Put(cp *ConnectionPool, db *sql.DB) {
	cp.ch <- db
}
func benchmarkConnectionPool(maxConn int) {
	timeNow := time.Now()
	var wg sync.WaitGroup

	cp := NewConnectionPool(10)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			db := Get(cp)
			_, err := db.Exec("SELECT sleep(0.01);")
			if err != nil {
				println("Benchmark Connection Pool", time.Since(timeNow).Round(time.Millisecond))
				panic(err)
			}
			Put(cp, db)
			wg.Done()
		}()
	}
	wg.Wait()
	println("Benchmark Connection Pool", time.Since(timeNow).String())
}

func benchmarkPlain1() {
	// simulate concurrent users by go routines
	timeNow := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			db := newConn()
			// db.SetMaxOpenConns(1)
			// db.SetMaxOpenConns(1)
			// db.SetMaxIdleConns(0)
			// db.SetConnMaxLifetime(0)
			// db.SetConnMaxIdleTime(0)
			_, err := db.Exec("SELECT sleep(0.01);")
			if err != nil {
				println("Benchmark Non Pool", time.Since(timeNow).Round(time.Millisecond))
				panic(err)
			}
			db.Close()
		}()
	}
	wg.Wait()
	println("Benchmark Non Pool", time.Since(timeNow).String())
}
func temp() {
	// benchmarkPlain()
	// benchmarkConnectionPool(10)
}
