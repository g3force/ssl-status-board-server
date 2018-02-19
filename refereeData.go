package main

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
