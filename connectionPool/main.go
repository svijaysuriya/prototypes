package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	// Replace this with a real DB driver import if needed.
	// For demonstration, we'll keep the MySQL driver placeholder.
	_ "github.com/go-sql-driver/mysql"
)

// Global variable to hold the single, configured database connection pool instance.
var dbPool *sql.DB

// initDBPool initializes and configures the *production-grade* built-in connection pool.
func initDBPool() {
	// IMPORTANT: *sql.Open does not establish any connections yet.
	// It only initializes the DB object and validation logic.
	db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/cricket_db")
	if err != nil {
		// Log and panic if the driver initialization fails.
		log.Fatal("Error opening database: ", err)
	}

	// 1. Production-Grade Pool Configuration (The Real Bounded Blocking Queue)
	// These settings control the actual underlying connection pool managed by the driver.

	// SetMaxOpenConns is the single most important setting: it bounds the pool size.
	// No more than 10 physical connections will ever be created.
	db.SetMaxOpenConns(10) // Equivalent to the max size of the Bounded Queue (N=10)

	// SetMaxIdleConns controls how many connections are kept open when idle.
	// This should generally be less than or equal to MaxOpenConns.
	db.SetMaxIdleConns(5)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	// This helps prevent stale connections. 1 hour is a common value.
	db.SetConnMaxLifetime(time.Hour)

	// SetConnMaxIdleTime sets the maximum amount of time a connection can be idle.
	// Use this if your server has aggressive idle timeouts.
	db.SetConnMaxIdleTime(time.Minute * 5)

	// 2. Test Connection Health (Critical for Production)
	// Ping the database to ensure a connection can be established immediately.
	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}

	dbPool = db
	fmt.Println("Production DB Pool successfully initialized and configured.")
}

// benchmarkProductionPool demonstrates the use of the properly configured built-in pool.
// The *sql.DB object is thread-safe, so we use it directly across all goroutines.
func benchmarkProductionPool(numQueries int) {
	timeNow := time.Now()
	var wg sync.WaitGroup

	// Use the single, globally configured pool
	pool := dbPool

	fmt.Printf("\n--- Benchmarking Production Pool (Max Open Conns: %d, Queries: %d) ---\n",
		pool.Stats().MaxOpenConnections, numQueries)

	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func(queryID int) {
			defer wg.Done()

			// 1. Acquire Connection (Internal to sql.DB)
			// db.QueryRow() or db.Exec() automatically acquires a connection from the internal pool.
			// If all 10 connections are busy, this goroutine will block until one is released.

			// Use a simple query that simulates work
			_, err := pool.Exec("SELECT sleep(0.01);")

			// 2. Connection Health/Error Check
			if err != nil {
				log.Printf("Query %d failed: %v", queryID, err)
				return
			}

			// 3. Release Connection (Internal to sql.DB)
			// The connection is automatically released back to the pool
			// when the context of the operation (Exec) is finished.

		}(i + 1)
	}
	wg.Wait()

	fmt.Printf("Benchmark Production Pool Time: %s\n", time.Since(timeNow).String())
	fmt.Printf("Pool Statistics at end: %+v\n", pool.Stats())
}

// benchmarkPlain demonstrates the high overhead of repeatedly opening and closing connections.
func benchmarkPlain(numQueries int) {
	timeNow := time.Now()
	var wg sync.WaitGroup

	fmt.Printf("\n--- Benchmarking Un-Pooled Connections (Queries: %d) ---\n", numQueries)

	for i := 0; i < numQueries; i++ {
		wg.Add(1)

		go func(queryID int) {
			defer wg.Done()

			// HIGH OVERHEAD: Opening a new connection for every request
			db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/cricket_db")
			if err != nil {
				log.Printf("Un-pooled query %d failed (Open): %v", queryID, err)
				return
			}

			// Ensure the connection is closed immediately after use
			defer db.Close()

			_, err = db.Exec("update teams set totalScore = 99 where teamNo = 1;")
			if err != nil {
				log.Printf("Error Un-pooled query %d failed (Exec): %v", queryID, err)
				panic(err)
				return
			}
		}(i + 1)
	}
	wg.Wait()

	fmt.Printf("Benchmark Un-Pooled Time: %s\n", time.Since(timeNow).String())
}

func main() {
	// The user's code had the plain benchmark running first, let's keep that structure.
	benchmarkPlain(152)

	// Initialize the single, production-grade connection pool
	// initDBPool()

	// Run the benchmark using the configured pool
	// benchmarkProductionPool(500)

	// Clean up the main pool when done
	if dbPool != nil {
		dbPool.Close()
	}
}
