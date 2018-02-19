package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
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
var proxyConfig ProxyConfig

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
		log.Println("Status provider tried to connect with wrong credentials:", user, password)
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

func loadCredentials() {
	for _, a := range proxyConfig.AuthCredentials {
		credentials = append(credentials, a.Username+":"+a.Password)
	}
}

func main() {

	configFile := flag.String("c", "proxy-config.yaml", "The config file to use")
	flag.Parse()

	proxyConfig = ReadProxyConfig(*configFile)
	log.Println("Proxy config:", proxyConfig)

	loadCredentials()

	go sendStatus()
	http.HandleFunc(proxyConfig.SubscribePath, serveStatusHandler)
	http.HandleFunc(proxyConfig.PublishPath, receiveStatusHandler)
	log.Println("Start listener on", proxyConfig.ListenAddress)
	log.Fatal(http.ListenAndServe(proxyConfig.ListenAddress, nil))
}
