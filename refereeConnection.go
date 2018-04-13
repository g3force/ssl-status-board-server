package main

import (
	"github.com/RoboCup-SSL/ssl-go-tools/sslproto"
	"log"
	"net"
	"github.com/golang/protobuf/proto"
)

func handleIncomingRefereeMessages() {
	refereeAddr := serverConfig.RefereeAddress
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
	if message.GameEvent != nil {
		if message.GameEvent.Originator != nil {
			referee.GameEvent.Originator.Team = message.GameEvent.Originator.Team.String()
			if message.GameEvent.Originator.BotId == nil {
				referee.GameEvent.Originator.BotId = -1
			} else {
				referee.GameEvent.Originator.BotId = int(*message.GameEvent.Originator.BotId)
			}
		} else {
			referee.GameEvent.Originator.Team = "TEAM_UNKNOWN"
			referee.GameEvent.Originator.BotId = -1
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