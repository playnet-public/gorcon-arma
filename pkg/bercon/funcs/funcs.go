package funcs

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"net"
	"strings"

	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
	"github.com/playnet-public/libs/log"
	"go.uber.org/zap"
)

//RconFuncs defines a common set of functions that is exported for external use
type RconFuncs struct {
	log    *log.Logger
	Client *rcon.Client
}

//New RconFuncs Instance
func New(log *log.Logger, c *rcon.Client) *RconFuncs {
	rf := new(RconFuncs)
	rf.log = log
	rf.Client = c
	return rf
}

//GetPlayers from Server
//TODO: This looks way to dirty, maybe look for better solutions
func (f RconFuncs) GetPlayers() ([]*rcon.Player, error) {
	r, w := io.Pipe()

	players := new(rcon.Players)
	quit := make(chan error)

	go f.scanForPlayers(players, r, quit)

	//Fetch Player List from RCon
	err := f.Client.Exec([]byte("players"), w)
	if err != nil {
		return nil, err
	}
	q := <-quit
	if q == nil {
		return players.GetAll(), nil
	}
	return nil, q
}

//GetBans from Server (legacy BattlEye Bans)
//TODO: This looks way to dirty, maybe look for better solutions
func (f RconFuncs) GetBans() ([]*rcon.Ban, error) {

	r, w := io.Pipe()

	bans := new(rcon.Bans)
	quit := make(chan error)

	go f.scanForBans(bans, r, quit)

	//Fetch Ban List from RCon
	err := f.Client.Exec([]byte("bans"), w)
	if err != nil {
		return nil, err
	}
	q := <-quit
	if q == nil {
		return bans.GetAll(), nil
	}
	return nil, q
}

//Ban the passed in Player for duration (in minutes)
func (f RconFuncs) Ban(p *rcon.Player, duration int64, reason string) error {
	cmd := fmt.Sprintf("addBan %s %d %s", p.ExtID, duration, reason)
	f.log.Debug("sending command", zap.String("cmd", cmd))
	err := f.Client.Exec([]byte(cmd), os.Stdout)
	if err != nil {
		return err
	}
	cmd = fmt.Sprintf("loadBans")
	f.log.Debug("sending command", zap.String("cmd", cmd))
	return f.Client.Exec([]byte(cmd), os.Stdout)
}

//MultiBan adds the array of Bans to the Server and does a reload
func (f RconFuncs) MultiBan(bans []*rcon.Ban) error {
	for _, ban := range bans {
		var duration int64
		if ban.Ends.IsZero() {
			duration = 10
		} else {
			dur := time.Until(ban.Ends)
			duration = int64(dur.Minutes())
		}
		cmd := fmt.Sprintf("addBan %s %d %s", ban.Descriptor, duration, ban.Reason)
		f.log.Debug("sending command", zap.String("cmd", cmd))
		err := f.Client.Exec([]byte(cmd), os.Stdout)
		if err != nil {
			return err
		}
	}
	cmd := fmt.Sprintf("loadBans")
	f.log.Debug("sending command", zap.String("cmd", cmd))
	return f.Client.Exec([]byte(cmd), os.Stdout)
}

//Kick the passed in Player
func (f RconFuncs) Kick(p *rcon.Player, reason string) error {
	cmd := fmt.Sprintf("kick %d %s", p.ID, reason)
	f.log.Debug("sending command", zap.String("cmd", cmd))
	return f.Client.Exec([]byte(cmd), os.Stdout)
}

//Message the passed in Player
func (f RconFuncs) Message(p *rcon.Player, msg string) error {
	cmd := fmt.Sprintf("say %d %s", p.ID, msg)
	f.log.Debug("sending command", zap.String("cmd", cmd))
	return f.Client.Exec([]byte(cmd), os.Stdout)
}

//ParsePlayerEvent and return Player, Event and Error
func (f RconFuncs) ParsePlayerEvent(s string) (p *rcon.Player, e rcon.PlayerEvent, err error) {
	e = rcon.PlayerEvent{}
	e.Type = -1
	e.Raw = s
	e.Timestamp = time.Now()
	p = new(rcon.Player)

	eventType := RegEx.Type.FindString(s)
	switch eventType {
	case "connected":
		e.Type = 0
	case "disconnected":
		e.Type = 3
	case "GUID:":
		e.Type = 1
	case "Verified":
		e.Type = 2
	case "RCon":
		e.Type = 7
	default:
		err = common.ErrUnknownEventType
		f.log.Error("unknown event type", zap.String("event", s), zap.Error(err))
		return
	}

	pid := -1
	pidM := RegEx.PlayerID.FindStringSubmatch(s)
	f.log.Debug("parsing playerID", zap.Strings("pid", pidM), zap.String("event", s))
	if len(pidM) > 1 {
		pid, err = strconv.Atoi(pidM[1])
		if err == nil {
			p.ID = pid
		}
	}
	guidM := RegEx.GUID.FindStringSubmatch(s)
	f.log.Debug("parsing guid", zap.Strings("guid", guidM), zap.String("event", s))
	if len(guidM) > 1 {
		p.ExtID = guidM[1]
	}
	if len(guidM) > 2 {
		err = common.ErrEscalationAttempt
		f.log.Error("multiple guid's in event", zap.Strings("guids", guidM), zap.String("event", s))
		switch e.Type {
		case 1:
			p.ExtID = guidM[len(guidM)-1]
		case 2:
			p.ExtID = guidM[1]
		}
	}
	if e.Type == 2 {
		nameA := RegEx.PlayerInfo.FindStringSubmatch(s)
		if len(nameA) > 2 {
			p.Name = nameA[2]
		}
	}
	netInf := RegEx.NetInf.FindString(s)
	f.log.Debug("parsing netInf", zap.String("netInf", netInf), zap.String("event", s))
	netInfA := strings.Split(netInf, ":")
	if len(netInfA) > 1 {
		p.IP = net.ParseIP(netInfA[0])
		p.Port = netInfA[1]
	}
	return
}

//ParseMessage and return it
func (f RconFuncs) ParseMessage(s string) *rcon.Message {
	m := new(rcon.Message)
	//TODO: parse event strings here
	m.Content = s
	return m
}
