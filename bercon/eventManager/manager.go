package eventManager

import (
	"sync"

	"fmt"

	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/rcon"
)

//EventTypes are mapping the event Type to their string values
var EventTypes = struct {
	Join       string
	Login      string
	Disconnect string
}{
	Join:       "join",
	Login:      "login",
	Disconnect: "disconnect",
}

//EventManager is responsible for handling Events
type EventManager struct {
	Events rcon.Events
	rwm    sync.RWMutex
}

//NewEventManager returns a new Manager Object
func NewEventManager() *EventManager {
	pm := new(EventManager)
	return pm
}

//Parse the Events List
func (pm *EventManager) Parse(data string) (rcon.Event, error) {
	ev := new(rcon.Event)
	//TODO: parse event strings here
	ev.Content = data
	return *ev, nil
}

//Add the passed in Event
func (pm *EventManager) Add(ev rcon.Event) error {
	glog.V(3).Infoln("Adding Event:", ev)
	pm.rwm.Lock()
	defer pm.rwm.Unlock()
	ev.ID = len(pm.Events)
	pm.Events = append(pm.Events, ev)
	return nil
}

//Get returns all events
func (pm *EventManager) Get() rcon.Events {
	pm.rwm.RLock()
	defer pm.rwm.RUnlock()
	return pm.Events
}

//GetNew returns all events after last(id)
func (pm *EventManager) GetNew(last int) (rcon.Events, error) {
	pm.rwm.RLock()
	defer pm.rwm.RUnlock()
	if len(pm.Events) < last {
		return nil, fmt.Errorf("can not fetch new events after %v: lenght missmatch", last)
	}
	return pm.Events[last:], nil
}
