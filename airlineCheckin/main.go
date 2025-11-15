package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

type Seat struct {
	id   int
	seat string
	name sql.NullString
}

type User struct {
	id   int
	name string
}

func connectToDb() {
	db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/airline")
	if err != nil {
		log.Fatal("error while connecting to mysql DB | ", err)
	}
	Db = db
}

func reserveEmptyForUser(userName string, wg *sync.WaitGroup) {
	defer wg.Done()

	tx, err := Db.Begin()
	if err != nil {
		fmt.Println("error while creating transaction, err =", err.Error())
		tx.Rollback()
		return
	}
	dbStmt, err := tx.Prepare("select id from seat where name is NULL ORDER BY id LIMIT 1 for update;")
	if err != nil {
		fmt.Println("error while preparing empty seat fetch stmt, err =", err.Error())
		tx.Rollback()
		return
	}
	defer dbStmt.Close()

	rows, err := dbStmt.Query()
	if err != nil {
		fmt.Println("error while executing empty seat fetch, err =  ", err.Error())
		tx.Rollback()
		return
	}
	var emptySeatId int
	if rows.Next() {
		err = rows.Scan(&emptySeatId)
		if err != nil {
			fmt.Println("error while Scan emptySeatId, err =  ", err.Error())
			tx.Rollback()
			return
		}
	}
	rows.Close()
	if emptySeatId != 0 {
		res, err := tx.Exec("update seat set name = ? where id = ?", userName, emptySeatId)
		if err != nil {
			fmt.Println(userName, " | open connection = ", Db.Stats().OpenConnections, " | inuse = ", Db.Stats().InUse, "error while updating the seat, err =  ", err.Error())
			tx.Rollback()
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 1 {
			fmt.Println("booked seat id ", emptySeatId, "  for -> ", userName)
		} else {
			fmt.Println("failed to book for => ", userName)
		}
	} else {
		fmt.Println("\n-----Attention------ All seats booked!! ", userName)
	}
	tx.Commit()

}
func makeBooking() {
	dbStmt, err := Db.Prepare("select * from user")
	if err != nil {
		fmt.Println("error while preparing user fetch stmt, err =  ", err.Error())
		return
	}

	rows, err := dbStmt.Query()
	if err != nil {
		fmt.Println("error while executing user fetch, err =  ", err.Error())
		return
	}
	var wg sync.WaitGroup
	users := make([]User, 0)
	for rows.Next() {
		var us User
		err = rows.Scan(&us.id, &us.name)
		if err != nil {
			fmt.Println("error while Scan user data, err =  ", err.Error())
			return
		}
		users = append(users, User{id: us.id, name: us.name})
	}
	rows.Close()
	for _, user := range users {
		wg.Add(1)
		go reserveEmptyForUser(user.name, &wg)

	}
	wg.Wait()
}

func printSeats() {
	dbStmt, err := Db.Prepare("select * from seat")
	if err != nil {
		fmt.Println("error while preparing seat fetch stmt, err =  ", err.Error())
		return
	}

	rows, err := dbStmt.Query()
	if err != nil {
		fmt.Println("error while executing user seat, err =  ", err.Error())
		return
	}
	count := 0
	defer rows.Close()

	for rows.Next() {
		var us Seat
		err = rows.Scan(&us.id, &us.seat, &us.name)
		if err != nil {
			fmt.Println("error while Scan seat data, err =  ", err.Error())
			return
		}
		count++
		if count%7 == 0 {
			fmt.Println()
		}
		fmt.Print(us.name.String, "\t| ")
	}
	fmt.Println()
}

func main() {
	startTime := time.Now()
	connectToDb()
	printSeats()
	makeBooking()
	printSeats()
	fmt.Println("for update time elasped = ", time.Since(startTime).String())
}
