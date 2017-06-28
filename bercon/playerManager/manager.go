package playerManager

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"regexp"

	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/rcon"
)

//PlayerManager is responsible for handling Players and their actions
type PlayerManager struct {
	Players rcon.Players
	Client  *rcon.Client
}

//NewPlayerManager returns a new Manager Object
func NewPlayerManager() *PlayerManager {
	pm := new(PlayerManager)
	return pm
}

//Refresh the Players List
//TODO: This looks way to dirty, maybe look for better solutions
func (pm *PlayerManager) Refresh() error {

	r, w := io.Pipe()

	var players rcon.Players
	quit := make(chan error)

	go scanForPlayers(players, r, quit)

	//Fetch Player List from RCon
	err := pm.Client.Exec([]byte("players"), w)
	if err != nil {
		return err
	}
	q := <-quit
	if q == nil {
		pm.Players = players
		return nil
	}
	return q
}

//Get returns all players
func (pm *PlayerManager) Get() rcon.Players {
	return pm.Players
}

func scanForPlayers(players rcon.Players, r io.ReadCloser, quit chan error) {
	reg, err := regexp.Compile(`(\d+)\s+(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d+\b)\s+(\d+)\s+([0-9a-fA-F]+)\(\w+\)\s([\S ]+)`)
	if err != nil {
		quit <- err
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		errUnable := fmt.Errorf("unable to parse player: %v", line)
		glog.V(3).Infoln("Applying Regex onto:", line)
		playerInfo := reg.FindStringSubmatch(line)
		if len(playerInfo) < 1 {
			continue
		}
		playerInfo = playerInfo[1:]
		fmt.Println("Player Matched:", playerInfo)
		id, err := strconv.Atoi(playerInfo[0])
		if err != nil {
			quit <- err
		}
		port, err := strconv.Atoi(playerInfo[2])
		if err != nil {
			quit <- err
		}
		ping, err := strconv.Atoi(playerInfo[3])
		if err != nil {
			quit <- err
		}
		ip := net.ParseIP(playerInfo[1])
		if ip == nil {
			quit <- errUnable
		}
		player := rcon.Player{
			ID:   id,
			Name: playerInfo[5],
			GUID: playerInfo[4],
			IP:   ip,
			Port: port,
			Ping: ping,
		}
		players = append(players, player)
	}
	if err := scanner.Err(); err != nil {
		quit <- err
	}
	quit <- nil
}

//Ban the passed in Player for duration (in minutes)
func (pm *PlayerManager) Ban(p rcon.Player, duration int64, reason string) error {
	return pm.Client.Exec([]byte(fmt.Sprint("addBan", p.GUID, duration, reason)), os.Stdout)
}

//Kick the passed in Player
func (pm *PlayerManager) Kick(p rcon.Player, reason string) error {
	return pm.Client.Exec([]byte(fmt.Sprint("kick", p.ID, reason)), os.Stdout)
}

//Message the passed in Player
func (pm *PlayerManager) Message(p rcon.Player, msg string) error {
	return pm.Client.Exec([]byte(fmt.Sprint("say", p.ID, msg)), os.Stdout)
}
