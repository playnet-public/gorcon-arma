package rcon

import (
	"net"
)

//PlayerManager is responsible for handling Players and their actions
type PlayerManager struct {
	Players Players
	Refresh refreshPlayers
	Get     getPlayers
	Ban     banPlayer
	Kick    kickPlayer
	Message messagePlayer
}

//Player represents an abstract rcon player
type Player struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	GUID string `json:"guid"`
	IP   net.IP `json:"ip"`
	Port int    `json:"port"`
	Ping int    `json:"ping"`
}

//Players is the Player List
type Players []Player

type refreshPlayers func() error
type getPlayers func() Players
type banPlayer func(p Player, duration int64, reason string) error
type kickPlayer func(p Player, reason string) error
type messagePlayer func(p Player, msg string) error

//NewPlayerManager returns a new Manager Object
func NewPlayerManager(
	refreshPlayers refreshPlayers,
	get getPlayers,
	ban banPlayer,
	kick kickPlayer,
	message messagePlayer,
) *PlayerManager {
	pm := new(PlayerManager)
	pm.Refresh = refreshPlayers
	pm.Get = get
	pm.Ban = ban
	pm.Kick = kick
	pm.Message = message
	return pm
}
