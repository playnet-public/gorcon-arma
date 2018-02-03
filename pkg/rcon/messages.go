package rcon

import (
	"bufio"
	"io"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
)

//MessageManager is responsible for handling Messages and their actions
type MessageManager struct {
	Messages Messages
	Parse    parseMessage
	Add      addMessage
	Get      getMessages
	GetNew   getNewMessages
}

//Message represents an abstract rcon event
type Message struct {
	ID        int       `json:"id"`
	Author    string    `json:"author"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

//Messages is the Message List
type Messages []Message

type parseMessage func(data string) (Message, error)
type addMessage func(p Message) error
type getMessages func() Messages
type getNewMessages func(last int) (Messages, error)

//NewMessageManager returns a new Manager Object
func NewMessageManager(
	parse parseMessage,
	add addMessage,
	get getMessages,
	getNew getNewMessages,
) *MessageManager {
	pm := new(MessageManager)
	pm.Parse = parse
	pm.Add = add
	pm.Get = get
	pm.GetNew = getNew
	return pm
}

//Listen to the passed in writer for messages
func (em *MessageManager) Listen(r io.Reader, errc chan error) {
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			ev, err := em.Parse(sc.Text())
			if err != nil {
				errc <- err
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "messages"})
			}
			if err := em.Add(ev); err != nil {
				errc <- err
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "messages"})
			}
		}
		if err := sc.Err(); err != nil {
			glog.Errorln("scanning for messages errored", err)
			errc <- err
			raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "messages"})
			return
		}
	}()
}
