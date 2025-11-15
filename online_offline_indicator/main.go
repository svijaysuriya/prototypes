package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
)

var Db *sql.DB

func handleHearBeat(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("user_id")
	if userId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("user_id required"))
		return
	}
	currentTime := time.Now()
	stmt, err := Db.Prepare("insert into user (user_id,last_timestamp) values (?,?) on duplicate key update last_timestamp = ?")
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error from db"))
		return
	}
	_, err = stmt.Exec(userId, currentTime, currentTime)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error from db"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hb updated!"))
}
func fetchOnlineOfflineStatus(w http.ResponseWriter, r *http.Request) {
	query := "select * from user"
	rows, err := Db.Query(query)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error from db"))
		return
	}
	currentTime := time.Now()
	type user struct {
		user_id        int
		last_timestamp time.Time
	}
	var us user
	var output string
	for rows.Next() {
		err = rows.Scan(&us.user_id, &us.last_timestamp)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error from db"))
			return
		}
		// fmt.Println("last usertimestamp : ", us.last_timestamp, "\tcurrentTime : ", currentTime, "\t currentTime.Sub(us.last_timestamp) = ", currentTime.Sub(us.last_timestamp))

		if currentTime.Sub(us.last_timestamp) <= time.Second*10 {
			output += fmt.Sprintf("%d : %v\n", us.user_id, "online")
		} else {
			output += fmt.Sprintf("%d : last seen %v\n", us.user_id, currentTime.Sub(us.last_timestamp).String())
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

func main() {
	connectToDb()
	// http.HandleFunc("/hb", handleHearBeat)
	// http.HandleFunc("/status", fetchOnlineOfflineStatus)
	// fmt.Println("Starting server in 8080")

	// err := http.ListenAndServe(":8080", nil)
	// if err != nil {
	// 	panic(err)
	// }

	// socket.io - http long polling fallback and websocket if possible
	// EngineIO Options for Heartbeat Testing
	serverEngineOptions := engineio.Options{
		PingTimeout:  20 * time.Second, // 20000 ms (20 seconds)
		PingInterval: 10 * time.Second, // 10000 ms (10 seconds)
	}

	// Initialize the Socket.IO Server with EngineIO Options
	server := socketio.NewServer(&serverEngineOptions)
	timeTaken := make(map[string]time.Time)
	// OnConnect handler for the default namespace "/"
	server.OnConnect("/", func(c socketio.Conn) error {
		fmt.Println("connected successfully:", c.ID())
		// Log the transport type and the namespace
		fmt.Println("Transport:", c.RemoteHeader().Get("transport"), "Namespace:", c.Namespace())
		timeTaken[c.ID()] = time.Now()
		return nil
	})

	// OnError handler
	server.OnError("/", func(c socketio.Conn, e error) {
		fmt.Println("Error occurred:", e)
	})

	// OnDisconnect handler
	server.OnDisconnect("/", func(c socketio.Conn, reason string) {
		fmt.Println("Client disconnected:", c.ID(), "Reason:", reason, "\tTime Taken to disconnect: ", time.Since(timeTaken[c.ID()]).String())
	})

	server.OnEvent("/", "hearbeat", func(conn socketio.Conn) {
		fmt.Println("received heartbeat from ", conn.ID())
		currentTime := time.Now()
		stmt, err := Db.Prepare("insert into user (user_id,last_timestamp) values (?,?) on duplicate key update last_timestamp = ?")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_, err = stmt.Exec(conn.ID(), currentTime, currentTime)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	})

	// Start the server in a goroutine
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	// Setup HTTP handlers
	// Serve the socket.io requests at the default path
	http.Handle("/socket.io/", server)

	// Optional: Serve a static file (e.g., an index.html with a client)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	http.HandleFunc("/status", fetchOnlineOfflineStatus)

	port := ":8000"
	log.Println("Serving Socket.IO at localhost" + port + "...")
	log.Fatal(http.ListenAndServe(port, nil))
}
func connectToDb() {
	db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/onOffIndicator?parseTime=true")
	if err != nil {
		panic(err)
	}
	Db = db
	fmt.Println("Connected to db")
}
