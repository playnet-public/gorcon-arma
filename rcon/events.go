package rcon

import (
	"bufio"
	"io"
	"time"

	"sync"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
)

//EventManager is responsible for handling Events
type EventManager struct {
	Events Events
	Parse  parseEvent
	Add    addEvent
	Get    getEvents
	GetNew getNewEvents
	//PlayerManager *PlayerManager
}

//Event represents an abstract rcon event
type Event struct {
	Source    string    `json:"src"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Raw       string    `json:"raw"`
}

//PlayerEventType is the player linked events type
type PlayerEventType int

//PlayerEventTypes list the various event types a player could be linked to
var PlayerEventTypes = struct {
	Connect    PlayerEventType
	Check      PlayerEventType
	Verified   PlayerEventType
	Disconnect PlayerEventType
	Chat       PlayerEventType
	Kick       PlayerEventType
	Ban        PlayerEventType
	Rcon       PlayerEventType
}{
	Connect:    0,
	Check:      1,
	Verified:   2,
	Disconnect: 3,
	Chat:       4,
	Kick:       5,
	Ban:        6,
	Rcon:       7,
}

//PlayerEvent describes events that can be linked to a player
type PlayerEvent struct {
	Source    string          `json:"src"`
	Type      PlayerEventType `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Raw       string          `json:"raw"`
}

//PlayerEvents provides a lockable PlayerEvents Array
type PlayerEvents struct {
	e []PlayerEvent
	sync.RWMutex
}

//Events is the Event List
type Events []Event

type parseEvent func(data string) (Event, error)
type addEvent func(p Event) error
type getEvents func() Events
type getNewEvents func(last int) (Events, error)

//NewEventManager returns a new Manager Object
func NewEventManager(
	parse parseEvent,
	add addEvent,
	get getEvents,
	getNew getNewEvents,
	//pm *PlayerManager,
) *EventManager {
	em := new(EventManager)
	em.Parse = parse
	em.Add = add
	em.Get = get
	em.GetNew = getNew
	//em.PlayerManager = pm
	return em
}

//Listen to the passed in writer for events
func (em *EventManager) Listen(r io.Reader, errc chan error) {
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			ev, err := em.Parse(sc.Text())
			if err != nil {
				errc <- err
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "events"})
			}
			if err := em.Add(ev); err != nil {
				errc <- err
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "events"})
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
