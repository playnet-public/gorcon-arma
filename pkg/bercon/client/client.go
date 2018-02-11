package client

import (
	"io"
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/connection"

	"os"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
)

//New creates a Client with given Config
func New(con rcon.Connection, cred rcon.Credentials) *Client {
	if con.KeepAliveTimer == 0 {
		con.KeepAliveTimer = 10 //TODO: Evaluate default value
	}

	return &Client{
		cfg:  con,
		cred: cred,
	}
}

// Connect opens the udp connection
func (c *Client) Connect() (err error) {
	c.Lock()
	defer c.Unlock()
	c.con = connection.New()
	err = c.con.Connect(c.cfg.Addr)
	if err != nil {
		return err
	}
	err = c.con.Login(c.cred.Password)
	if err != nil {
		return err
	}
	return nil
}

//Connect opens a new Connection to the Server
func (c *Client) ConnectOld(q chan error) error {
	if !c.looping {
		c.looping = true
		c.init = true
		c.exit = false

		q <- common.ErrConnected
	}
	return nil
}

func (c *Client) loop() (err error) {
	wd := make(chan error)
	rd := make(chan error)

	go c.writerLoop(wd, c.cmdChan)
	go c.readerLoop(rd)

	select {
	case d := <-rd:
		c.looping = false
		glog.V(2).Infoln("reader exited with error:", d)
		return d
	case d := <-wd:
		c.looping = false
		glog.V(2).Infoln("writer exited with error:", d)
		return d
	}
}

//Loop for the client to remain active and process events
func (c *Client) Loop(q chan error) error {
	go func() {
		for {
			if !c.looping || !c.init {
				if c.exit == true {
					return
				}
				if err := c.Connect(); err != nil {
					glog.V(2).Info(err)
					//TODO: Add Reconnect Time Setting
					time.Sleep(time.Second * 3)
				}
				continue
			}
			q <- c.loop()
		}
	}()
	return nil
}

//Disconnect the Client
func (c *Client) Disconnect() error {
	c.exit = true
	return c.con.Close()
}

//Exec adds given cmd to command queue
func (c *Client) Exec(cmd []byte, resp io.WriteCloser) error {
	c.cmdChan <- protocol.Transmission{Command: cmd, WriteCloser: resp}
	return nil
}

//AttachEvents enables the event listener
//and returns a writer containing the event stream
func (c *Client) AttachEvents(w io.Writer) error {
	c.eventWriter.Lock()
	c.eventWriter.Writer = w
	c.eventWriter.Unlock()
	return nil
}

//AttachChat enables the chat listener
//and returns a writer containing the chat stream
func (c *Client) AttachChat(w io.Writer) error {
	c.chatWriter.Lock()
	c.chatWriter.Writer = w
	c.chatWriter.Unlock()
	return nil
}

func (c *Client) handleResponse(seq uint32, response []byte, last bool) {
	glog.V(6).Infoln("Handling Response:", response, string(response))
	trm, ex := c.con.GetTransmission(seq)
	if !ex {
		if len(response) == 0 {
			se := c.con.Sequence()
			if se == seq {
				glog.V(3).Infoln("Received KeepAlive Pingback")
				c.con.AddPingback()
			}
		} else {
			glog.Warningf("No Entry in cmdMap for: %v - (%v)", string(response), response)
		}
	} else {
		trail := []byte("\n")
		if !last {
			trail = []byte{}
		}
		trm.Response = append(trm.Response, response...)
		trm.Response = append(trm.Response, trail...)
		if last {
			if trm.WriteCloser != nil {
				glog.V(4).Infoln("Writing", string(trm.Response), "to output")
				trm.WriteCloser.Write(trm.Response)
				if trm.WriteCloser != os.Stderr && trm.WriteCloser != os.Stdout {
					err := trm.WriteCloser.Close()
					if err != nil {
						glog.Errorln(err)
						raven.CaptureError(err, map[string]string{"app": "rcon", "module": "client"})
					}
				}
			}

			//TODO: Evaluate if this is required
			go func(c *Client, seq uint32) {
				c.con.DeleteTransmission(seq)
			}(c, seq)
		}
	}
}
