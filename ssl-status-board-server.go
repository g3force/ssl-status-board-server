package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"encoding/json"
	"fmt"
	"time"
	"net/url"
	"encoding/base64"
	"flag"
)

const maxDatagramSize = 8192

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

var config Config

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	sendDataToWebSocket(conn)
}

func sendDataToWebSocket(conn *websocket.Conn) {
	for {
		b, err := json.Marshal(referee)
		if err != nil {
			fmt.Println("Marshal error:", err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Println(err)
			return
		}

		time.Sleep(config.SendingInterval)
	}
}

func broadcastToProxy() error {
	u := url.URL{Scheme: config.ServerProxy.Scheme, Host: config.ServerProxy.Address, Path: config.ServerProxy.Path}
	log.Printf("connecting to %s", u.String())

	auth := []byte(config.ServerProxy.User + ":" + config.ServerProxy.Password)
	authBase64 := base64.StdEncoding.EncodeToString(auth)

	requestHeader := http.Header{}
	requestHeader.Set("Authorization", "Basic "+authBase64)
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), requestHeader)
	if err != nil {
		return err
	}
	defer conn.Close()

	sendDataToWebSocket(conn)
	return nil
}

func handleServerProxy() {
	for {
		err := broadcastToProxy()
		log.Println("Disconnected from proxy ", err)
		if err != nil {
			time.Sleep(config.ServerProxy.ReconnectInterval)
		}
	}
}

func main() {

	configFile := flag.String("config", "config.yaml", "The config file to use")
	flag.Parse()

	config = ReadConfig(*configFile);
	log.Println(config)

	go handleIncomingRefereeMessages()

	if config.ServerProxy.Enabled {
		go handleServerProxy()
	}

	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/ssl-status", statusHandler)
	log.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}
