package rcon

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
)

//Funcs are the abstract Interface for wrapping Player related functions
type Funcs interface {
	GetPlayers() ([]*Player, error)
	GetBans() ([]*Ban, error)
	Ban(p *Player, duration int64, reason string) error
	MultiBan(bans []*Ban) error
	Kick(p *Player, reason string) error
	Message(p *Player, msg string) error
	ParsePlayerEvent(s string) (p *Player, e PlayerEvent, err error)
	ParseMessage(s string) *Message
}

//PlayerManager is responsible for handling Players and their actions
type PlayerManager struct {
	Players    Players
	Funcs      Funcs
	BanManager *BanManager
}

//Player represents an abstract rcon player
type Player struct {
	ID     int          `json:"id"`
	Name   string       `json:"name"`
	ExtID  string       `json:"extID"`
	IP     net.IP       `json:"ip"`
	Port   string       `json:"port"`
	Ping   string       `json:"ping"`
	Events PlayerEvents `json:"events"`
}

//Players is the Player List
type Players struct {
	p []*Player
	sync.RWMutex
}

//Add Player to Players
//WARN: This function seems to cause panics on empty Players Array
//TODO: Check this function for usablility (and need)
func (p *Players) Add(player *Player) {
	p.Lock()
	defer p.Unlock()
	if pl := p.p[player.ID]; pl != nil {
		glog.Warningf("Player already exists on index %v: %v - Overwriting with %v", player.ID, pl, player)
	}
	p.p[player.ID] = player
}

//Append to Players
func (p *Players) Append(player *Player) {
	p.Lock()
	defer p.Unlock()
	p.p = append(p.p, player)
}

//Remove Player from Players
//NOTE: This action could be expensive and is not finished yet. All Players get re-orderd but their own ID's stay in an inconsistent state
// It would be required to itterate over all players and reset the ID's which would block the whole array for quite some time
// For Player Disconnects we may need a different solution
func (p *Players) Remove(id int) {
	p.Lock()
	defer p.Unlock()
	if pl := p.p[id]; pl == nil {
		glog.Warningf("Player does not exist at index:", id)
		return
	}
	p.p = append(p.p[:id], p.p[id+1:]...)
}

//Get Player by ID
func (p *Players) Get(id int) *Player {
	p.RLock()
	defer p.RUnlock()
	pl := p.p[id]
	if pl != nil {
		if pl.ID != id {
			glog.Errorf("Player Array ID Mismatch: Index(%v) - PlayerID(%v)", id, pl.ID)
			return nil
		}
		return pl
	}
	glog.Errorln("No Player at Index", id)
	return pl
}

//GetAll Players
func (p *Players) GetAll() []*Player {
	p.RLock()
	defer p.RUnlock()
	return p.p
}

//SetAll Players
//WARN: This action overwrites the entire Players Array with new data which might be both expensive and destructive
//All PlayerEvents might get lost
func (p *Players) SetAll(pl []*Player) {
	p.Lock()
	defer p.Unlock()
	p.p = pl
}

//NewPlayerManager returns a new Manager Object
func NewPlayerManager(pf Funcs, bm *BanManager) *PlayerManager {
	pm := new(PlayerManager)
	pm.Funcs = pf
	pm.BanManager = bm
	return pm
}

//AddPlayer to the PlayerManager
func (pm *PlayerManager) AddPlayer(p *Player) error {
	pm.Players.Add(p)
	return nil
}

//RemovePlayer to the PlayerManager (this is a quite expensive task which should not be triggered on every disconnect)
func (pm *PlayerManager) RemovePlayer(id int) error {
	pm.Players.Remove(id)
	return nil
}

//GetPlayers from PlayerManager
func (pm *PlayerManager) GetPlayers() []*Player {
	return pm.Players.GetAll()
}

//GetPlayer from PlayerManager
func (pm *PlayerManager) GetPlayer(id int) *Player {
	return pm.Players.Get(id)
}

//KickByID is a helper function for easily generating kick events by PlayerID
func (pm *PlayerManager) KickByID(id int, reason string) error {
	p := new(Player)
	p.ID = id
	return pm.Funcs.Kick(p, reason)
}

//AddEvent pushes the passed in PlayerEvent to the Player Object
func (p *Player) AddEvent(e PlayerEvent) {
	p.Events.Lock()
	defer p.Events.Unlock()
	p.Events.e = append(p.Events.e, e)
}

//Listen to the passed in writer for PlayerEvents
//TODO: Refactor this function to be less dirty
func (pm *PlayerManager) Listen(r io.Reader, errc chan error) {
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			pl, ev, err := pm.Funcs.ParsePlayerEvent(sc.Text())
			if err != nil {
				//errc <- err
				//glog.Errorln(err)
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "events"})
				continue
			}
			if ev.Type == 1 {
				fmt.Println("triggering ban check due to event", sc.Text())
				if isBanned, ban := pm.BanManager.Check(pl.ExtID); isBanned {
					//TODO: This approach will solve a problem with ArmA and Ban (with reload) while a kick might be best
					//Either find a better solution here or wrap this correctly to remain generic
					//If we stay with time banning, it might be good to add a config value for the time to ban (ban retention)
					pm.Funcs.Ban(pl, 10, ban.Reason)
					continue
				}
			} else {
				//glog.Warningln("Player Event without ExtID was skipped", sc.Text())
			}
			pl.AddEvent(ev)
			if ev.Type == 2 {
				go func() {
					time.Sleep(time.Second * 10)
					pm.Funcs.Message(pl, "GoRcon check completed. BanStatus: OK!")
					pm.Funcs.Message(pl, fmt.Sprintf("Welcome %s", pl.Name))
				}()
			}
		}
		if err := sc.Err(); err != nil {
			glog.Errorln("scanning for events errored", err)
			errc <- err
			raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "events"})
			return
		}
	}()
}

//CheckPlayers requests an up-to-date PlayerList and checks all players against the BanManager
func (pm *PlayerManager) CheckPlayers() error {
	glog.V(2).Infoln("Checking all Players")
	players, err := pm.Funcs.GetPlayers()
	var bans []*Ban
	if err != nil {
		return err
	}
	for _, p := range players {
		is, ban := pm.BanManager.Check(p.ExtID)
		if is {
			bans = append(bans, &Ban{Descriptor: p.ExtID, Reason: ban.Reason})
		}
	}
	return pm.Funcs.MultiBan(bans)
}
