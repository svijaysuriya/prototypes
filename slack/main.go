package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
)

var Db *sql.DB
var wsMap map[int64]*socketio.Conn

type User struct {
	Id            int64     `json:"id"`
	UserName      string    `json:"userName"`
	LastTimestamp time.Time `json:"lastTimestamp"`
}

// Channel represents a chat channel or group
type Channel struct {
	ChannelID   int64  `json:"channel_id" db:"channel_id"`
	ChannelType string `json:"channel_type" db:"channel_type"`
	ChannelName string `json:"channel_name" db:"channel_name"`
}

// Membership represents the link between users and channels
type Membership struct {
	MembershipID int64 `json:"membership_id" db:"membership_id"`
	ChannelID    int64 `json:"channel_id" db:"channel_id"`
	UserID       int64 `json:"user_id" db:"user_id"`
	// LastReadAt   time.Time `json:"last_read_at" db:"last_read_at"` // to store the unread messages
}

// Message represents an individual message sent in a channel
type Message struct {
	MessageID int64     `json:"message_id" db:"message_id"`
	SenderID  int64     `json:"sender_id" db:"sender_id"`
	ChannelID int64     `json:"channel_id" db:"channel_id"`
	Msg       string    `json:"msg" db:"msg"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	userName := r.PathValue("userName")
	row := Db.QueryRow("select id, userName from user where userName = ?", userName)
	if row.Err() != nil {
		fmt.Println("error while fetch user, err =  ", row.Err())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}
	var user User
	row.Err()
	err := row.Scan(&user.Id, &user.UserName)
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			fmt.Println("error while scan into variable, err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("scan try again")))
			return
		}
	}
	if user.UserName == "" {
		// create user
		fmt.Println("creating a new user")
		currTime := time.Now()
		res, err := Db.Exec("insert into user (userName,last_timestamp) values (?,?)", userName, currTime)
		if err != nil {
			fmt.Println("insert error = ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("try again")))
			return
		}
		user.Id, _ = res.LastInsertId()
		user.UserName = userName
	}
	fmt.Println("user(id, userName, last_timestamp) = ", user)
	// time.Sleep(5 * time.Second)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	userBytes, _ := json.Marshal(user)
	_, err = w.Write(userBytes)
	if err != nil {
		fmt.Println("error while writing response to client, err =  ", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("try again")))
	}
}

func createChannel(w http.ResponseWriter, r *http.Request) {
	senderName := r.PathValue("senderName")
	receiverName := r.PathValue("receiverName")

	// check whether sender and. receiver already have a channel or not
	row := Db.QueryRow("select id from user where userName = ?", senderName)
	if row.Err() != nil {
		fmt.Println("name sender fetch error while fetch user, err =  ", row.Err())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}
	var sender User
	err := row.Scan(&sender.Id)
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			fmt.Println("sender error while scan into variable, err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("scan try again")))
			return
		}
	}

	row = Db.QueryRow("select id from user where userName = ?", receiverName)
	if row.Err() != nil {
		fmt.Println("name receiver fetch error while fetch user, err =  ", row.Err())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}
	var receiver User
	err = row.Scan(&receiver.Id)
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			fmt.Println("receiver error while scan into variable, err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("scan try again")))
			return
		}
	}
	query := `SELECT c.channel_id, c.channel_type, c.channel_name
	FROM channel c
	JOIN membership m1 ON c.channel_id = m1.channel_id
	JOIN membership m2 ON c.channel_id = m2.channel_id
	WHERE m1.user_id = ?
	  AND m2.user_id = ?;`
	rows, err := Db.Query(query, sender.Id, receiver.Id)
	if err != nil {
		fmt.Println("membership check error err =  ", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("try again")))
		return
	}
	var ch Channel
	for rows.Next() {
		err = rows.Scan(&ch.ChannelID, &ch.ChannelType, &ch.ChannelName)
		if err != nil {
			fmt.Println("channel extract error err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("try again")))
			return
		}
	}
	rows.Close()

	output := make([]Message, 0)
	if ch.ChannelID == 0 {
		// create a channel b/w them
		result, err := Db.Exec("insert into channel (channel_type,channel_name) values (?,?)", "DM", senderName+"_"+receiverName)
		if err != nil {
			fmt.Println("failed to create channel b/w err =  ", senderName, receiverName, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("fetch try again")))
			return
		}
		channelId, err := result.LastInsertId()
		if err != nil {
			fmt.Println("failed to created channelId b/w err =  ", senderName, receiverName, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("fetch try again")))
			return
		}
		var channelCreate Message
		channelCreate.ChannelID = channelId
		channelCreate.Msg = "channel created b/w you and " + receiverName

		// create membership for them
		result, err = Db.Exec("insert into message (sender_id ,channel_id,msg,created_at) values (?,?,?,?)", sender.Id, channelId, channelCreate.Msg, time.Now())
		if err != nil {
			fmt.Println("failed to insert sender membership channel b/w err =  ", senderName, receiverName, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("fetch try again")))
			return
		}
		channelCreate.MessageID, _ = result.LastInsertId()
		output = append(output, channelCreate)
		go sendMessageOverWebSocket([]Membership{{ChannelID: channelId,
			UserID: receiver.Id}}, channelCreate.Msg, senderName)
		// create membership for them
		_, err = Db.Exec("insert into membership (channel_id,user_id) values (?,?)", channelId, sender.Id)
		if err != nil {
			fmt.Println("failed to insert sender membership channel b/w err =  ", senderName, receiverName, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("fetch try again")))
			return
		}
		_, err = Db.Exec("insert into membership (channel_id,user_id) values (?,?)", channelId, receiver.Id)
		if err != nil {
			fmt.Println("failed to insert receiver membership channel b/w err =  ", senderName, receiverName, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("fetch try again")))
			return
		}
	} else {
		// read last 10 message and send them
		query := `SELECT * from message where channel_id = ? order by created_at desc limit 10`
		rows, err := Db.Query(query, ch.ChannelID)
		if err != nil {
			fmt.Println("membership check error err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("try again")))
			return
		}
		for rows.Next() {
			var chMsg Message
			err = rows.Scan(&chMsg.MessageID, &chMsg.SenderID, &chMsg.ChannelID, &chMsg.Msg, &chMsg.CreatedAt)
			if err != nil {
				fmt.Println("channel extract error err =  ", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(string("try again")))
				return
			}
			output = append(output, chMsg)
		}
		rows.Close()
	}
	sender.UserName = senderName
	receiver.UserName = receiverName

	fmt.Println("sender = ", sender, "\treceiver = ", receiver)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputBytes, _ := json.Marshal(output)
	_, err = w.Write(outputBytes)
	if err != nil {
		fmt.Println("create channel error while writing response to client, err =  ", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("try again")))
	}
}

type MessageRequest struct {
	Msg string `json:"msg"`
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ip address", r.RemoteAddr)
	fmt.Println("X-Forwarded-For", r.Header.Get("X-Forwarded-For"))
	senderId := r.PathValue("senderId")
	channelId := r.PathValue("channelId")
	currentTime := time.Now()

	var req MessageRequest

	// Decode JSON body into struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received message: %s\n", req.Msg)

	// since it is a enterprise application first persit the chat
	_, err = Db.Exec("insert into message (sender_id,channel_id,msg,created_at) values (?,?,?,?)", senderId, channelId, req.Msg, currentTime)
	if err != nil {
		fmt.Println("failed to insert message into DB, err =  ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}

	// check whether sender and. receiver already have a channel or not
	rows, err := Db.Query("select * from membership where channel_id = ?", channelId)
	if err != nil {
		fmt.Println("fetch user in the given channel, err =  ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}

	senderIdInt, _ := strconv.Atoi(senderId)
	var members []Membership
	for rows.Next() {
		var member Membership
		err = rows.Scan(&member.MembershipID, &member.ChannelID, &member.UserID)
		if err != nil {
			fmt.Println("member extract for a channel error err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("try again")))
			return
		}
		if int(member.UserID) != senderIdInt {
			members = append(members, member)
		}
	}

	row := Db.QueryRow("select userName from user where id = ?", senderId)
	if row.Err() != nil {
		fmt.Println("error while fetch user, err =  ", row.Err())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("fetch try again")))
		return
	}
	var userName string
	row.Err()
	err = row.Scan(&userName)
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			fmt.Println("error while scan into variable, err =  ", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(string("scan try again")))
			return
		}
	}

	fmt.Println("members = ", members)
	sendMessageOverWebSocket(members, req.Msg, userName)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputBytes, _ := json.Marshal(req)
	_, err = w.Write(outputBytes)
	if err != nil {
		fmt.Println("create channel error while writing response to client, err =  ", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(string("try again")))
	}
}

func sendMessageOverWebSocket(members []Membership, msg string, userName string) {
	for _, member := range members {
		wsServer, present := wsMap[member.UserID]
		if present {
			(*wsServer).Emit("realtime", userName+":"+msg)
		}
	}
}
func connectToDb() {
	db, err := sql.Open("mysql", "root:localhost@tcp(localhost:3306)/slack?parseTime=true")
	if err != nil {
		log.Fatal("error while connecting to mysql DB | ", err)
	}
	Db = db
}

func connectToWebsocket() *socketio.Server {
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

	server.OnEvent("/", "hearbeat", func(conn socketio.Conn, msg interface{}) {
		fmt.Println("received heartbeat from ", msg)
		fmt.Println(msg)
		currentTime := time.Now()
		if msg != nil {
			userIdPlusUsername := msg.(string)
			splitArr := strings.Split(userIdPlusUsername, ",")
			userId, _ := strconv.Atoi(splitArr[0])
			wsMap[int64(userId)] = &conn
			fmt.Println("inserting for id, currTime => ", splitArr[0], currentTime)
			stmt, err := Db.Prepare("insert into user (id,userName,last_timestamp) values (?,?,?) on duplicate key update last_timestamp = ?")
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			_, err = stmt.Exec(splitArr[0], splitArr[1], currentTime, currentTime)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}

	})

	return server
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

	var us User
	output := make(map[string]bool, 0)
	for rows.Next() {
		err = rows.Scan(&us.Id, &us.UserName, &us.LastTimestamp)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error from db"))
			return
		}
		fmt.Println("last usertimestamp : ", us.LastTimestamp, "\tcurrentTime : ", currentTime, "\t currentTime.Sub(us.LastTimestamp) = ", currentTime.Sub(us.LastTimestamp))

		if currentTime.Sub(us.LastTimestamp) <= time.Second*10 {
			output[us.UserName] = true
		} else {
			output[us.UserName] = false
		}
	}
	outputByte, _ := json.Marshal(output)
	w.WriteHeader(http.StatusOK)
	w.Write(outputByte)
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	http.HandleFunc("/createUser/{userName}", createUser)
	http.HandleFunc("/channel/{senderName}/{receiverName}", createChannel)
	http.HandleFunc("/message/{senderId}/{channelId}", sendMessage)

	wsMap = make(map[int64]*socketio.Conn, 0)
	connectToDb()
	wsserver := connectToWebsocket()
	// Start the server in a goroutine
	go func() {
		if err := wsserver.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer wsserver.Close()

	// Setup HTTP handlers
	// Serve the socket.io requests at the default path
	http.Handle("/socket.io/", wsserver)
	http.HandleFunc("/status", fetchOnlineOfflineStatus)

	fmt.Println("starting web socket and slack server on port 4444")
	err := http.ListenAndServe(":4444", nil)
	if err != nil {
		log.Fatal(err)
	}
}
