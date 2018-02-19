package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"encoding/json"
	"fmt"
	"time"
	"github.com/RoboCup-SSL/ssl-go-tools/sslproto"
	"net"
	"github.com/golang/protobuf/proto"
)

const maxDatagramSize = 8192

type Team struct {
	Name            string
	Goals           int
	RedCards        int
	YellowCards     int
	YellowCardTimes []uint32
	Timeouts        int
	TimeoutTime     int
}

type Stage struct {
	Name     string
	TimeLeft int
}

type Command struct {
	Name string
}

type Originator struct {
	Team  string
	BotId int
}

type GameEvent struct {
	Type       string
	Originator Originator
	Message    string
}

type Referee struct {
	Stage      Stage
	Command    Command
	TeamYellow Team
	TeamBlue   Team
	GameEvent  GameEvent
}

var referee = Referee{Stage{"NORMAL_FIRST_HALF_PRE", 1000}, Command{"HALT"},
	Team{"yellow", 5, 0, 1, []uint32{5000}, 1, 10000},
	Team{"blue", 1, 1, 3, []uint32{}, 2, 20000},
	GameEvent{"UNKNOWN", Originator{"UNKNOWN", -1}, "Custom message"}}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	return true
}

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

func refereeHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		b, err := json.Marshal(referee)
		if err != nil {
			fmt.Println("error:", err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Println(err)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}

func handleIncomingRefereeMessages() {
	refereeAddr := "224.5.23.1:10003"
	err, refereeListener := openRefereeConnection(refereeAddr)
	if err != nil {
		log.Println("Could not connect to ", refereeAddr)
	}

	lastCommandId := uint32(100000000)
	for {
		data := make([]byte, maxDatagramSize)
		n, _, err := refereeListener.ReadFromUDP(data)
		if err != nil {
			log.Fatal("ReadFromUDP failed:", err)
		}

		message, err := parseRefereeMessage(data[:n])
		if err != nil {
			log.Print("Could not parse referee message: ", err)
		} else {
			saveRefereeMessageFields(message)

			if *message.CommandCounter != lastCommandId {
				log.Println("Received referee message:", message)
				lastCommandId = *message.CommandCounter
			}
		}
	}
}

func saveRefereeMessageFields(message *sslproto.SSL_Referee) {
	referee.Stage.Name = message.Stage.String()
	if message.StageTimeLeft != nil {
		referee.Stage.TimeLeft = int(*message.StageTimeLeft)
	} else {
		referee.Stage.TimeLeft = 0
	}
	referee.Command.Name = message.Command.String()
	referee.TeamYellow = mapTeam(message.Yellow)
	referee.TeamBlue = mapTeam(message.Blue)
	referee.GameEvent.Originator.Team = message.GameEvent.Originator.Team.String()
	if message.GameEvent != nil {
		if message.GameEvent.Originator.BotId == nil {
			referee.GameEvent.Originator.BotId = -1
		} else {
			referee.GameEvent.Originator.BotId = int(*message.GameEvent.Originator.BotId)
		}
		if message.GameEvent.Message == nil {
			referee.GameEvent.Message = ""
		} else {
			referee.GameEvent.Message = *message.GameEvent.Message
		}
		referee.GameEvent.Type = message.GameEvent.GameEventType.String()
	}
}

func mapTeam(teamInfo *sslproto.SSL_Referee_TeamInfo) (team Team) {
	team.Name = *teamInfo.Name
	team.Goals = int(*teamInfo.Score)
	team.YellowCards = int(*teamInfo.YellowCards)
	team.RedCards = int(*teamInfo.RedCards)
	team.YellowCardTimes = teamInfo.YellowCardTimes
	team.Timeouts = int(*teamInfo.Timeouts)
	team.TimeoutTime = int(*teamInfo.TimeoutTime)
	return
}

func openRefereeConnection(refereeAddr string) (err error, refereeListener *net.UDPConn) {
	addr, err := net.ResolveUDPAddr("udp", refereeAddr)
	if err != nil {
		log.Fatal(err)
	}
	refereeListener, err = net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		log.Fatal("could not connect to ", refereeAddr)
	}
	refereeListener.SetReadBuffer(maxDatagramSize)
	log.Printf("Listening on %s", refereeAddr)
	return
}

func parseRefereeMessage(data []byte) (message *sslproto.SSL_Referee, err error) {
	message = new(sslproto.SSL_Referee)
	err = proto.Unmarshal(data, message)
	return
}

func main() {
	go handleIncomingRefereeMessages()
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/referee", refereeHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
