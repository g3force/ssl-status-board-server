package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"os"
	"bufio"
	"flag"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

type WsMessage struct {
	messageType int
	data        []byte
}

var statusChannel = make(chan WsMessage, 100)
var clientConnections []*websocket.Conn
var statusProviderConnected = false
var credentials []string

func receiveStatusHandler(w http.ResponseWriter, r *http.Request) {

	user, password, ok := r.BasicAuth()
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("You have to authenticate yourself."))
		log.Println("Status provider tried to connect without credentials")
		return
	}
	if !validCredentials(user, password) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Your credentials are invalid."))
		log.Println("Status provider tried to connect with wrong credentials: ", user, password)
		return
	}

	if statusProviderConnected {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("There is already a status provider connected!"))
		log.Println("Another status provider tried to connect")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer disconnectStatusProvider(conn)

	log.Println("Status provider connected")

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		statusChannel <- WsMessage{messageType, p}
	}
}

func validCredentials(user string, password string) bool {
	credential := user + ":" + password
	for _, c := range credentials {
		if c == credential {
			return true
		}
	}
	return false
}

func disconnectStatusProvider(conn *websocket.Conn) {
	log.Println("Status provider disconnected")
	statusProviderConnected = false
	conn.Close()
}

func sendStatus() {
	for {
		wsMsg := <-statusChannel

		for _, conn := range clientConnections {
			if err := conn.WriteMessage(wsMsg.messageType, wsMsg.data); err != nil {
				log.Println(err)
				clientConnections = remove(clientConnections, conn)
				conn.Close()
			}
		}
	}
}

func remove(in []*websocket.Conn, conn *websocket.Conn) (out []*websocket.Conn) {
	out = []*websocket.Conn{}
	for _, c := range in {
		if conn != c {
			out = append(out, c)
		}
	}
	return
}

func serveStatusHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	clientConnections = append(clientConnections, conn)

	log.Printf("Client connected, now %d clients.\n", len(clientConnections))
}

func loadCredentials(authFile string) {
	file, err := os.Open(authFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		credentials = append(credentials, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {

	listenAddress := flag.String("a", ":4202", "The listen address ( [network]:port )")
	authFile := flag.String("f", "auth.conf", "A file containing valid basic auth credentials, one per line")
	flag.Parse()

	loadCredentials(*authFile)

	go sendStatus()
	http.HandleFunc("/ssl-status", serveStatusHandler)
	http.HandleFunc("/ssl-status-receive", receiveStatusHandler)
	log.Println("Start listener on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
