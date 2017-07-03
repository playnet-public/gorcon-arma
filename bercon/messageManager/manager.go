package messageManager

import (
	"sync"

	"fmt"

	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/rcon"
)

//MessageTypes are mapping the event Type to their string values
var MessageTypes = struct {
	Direct string
	Global string
	Side   string
}{
	Direct: "direct",
	Global: "global",
	Side:   "side",
}

//MessageManager is responsible for handling Messages
type MessageManager struct {
	Messages rcon.Messages
	rwm      sync.RWMutex
}

//NewMessageManager returns a new Manager Object
func NewMessageManager() *MessageManager {
	pm := new(MessageManager)
	return pm
}

//Parse the Messages List
func (pm *MessageManager) Parse(data string) (rcon.Message, error) {
	ev := new(rcon.Message)
	//TODO: parse event strings here
	ev.Content = data
	return *ev, nil
}

//Add the passed in Message
func (pm *MessageManager) Add(ev rcon.Message) error {
	glog.V(3).Infoln("Adding Message:", ev)
	pm.rwm.Lock()
	defer pm.rwm.Unlock()
	ev.ID = len(pm.Messages) - 1
	pm.Messages = append(pm.Messages, ev)
	return nil
}

//Get returns all messages
func (pm *MessageManager) Get() rcon.Messages {
	pm.rwm.RLock()
	defer pm.rwm.RUnlock()
	return pm.Messages
}

//GetNew returns all messages after last(id)
func (pm *MessageManager) GetNew(last int) (rcon.Messages, error) {
	pm.rwm.RLock()
	defer pm.rwm.RUnlock()
	if len(pm.Messages) < last {
		return nil, fmt.Errorf("can not fetch new messages after %v: lenght missmatch", last)
	}
	return pm.Messages[last:], nil
}
