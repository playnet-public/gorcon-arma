package rcon

import (
	"bufio"
	"io"
	"time"

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
}

//Event represents an abstract rcon event
type Event struct {
	ID        int       `json:"id"`
	Source    string    `json:"src"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
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
) *EventManager {
	pm := new(EventManager)
	pm.Parse = parse
	pm.Add = add
	pm.Get = get
	pm.GetNew = getNew
	return pm
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
