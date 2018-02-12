package client

import (
	"io"
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/connection"
	"github.com/playnet-public/libs/log"
	"go.uber.org/zap"

	"os"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
)

//New creates a Client with given Config
func New(log *log.Logger, con rcon.Connection, cred rcon.Credentials) *Client {
	if con.KeepAliveTimer == 0 {
		con.KeepAliveTimer = 10 //TODO: Evaluate default value
	}

	return &Client{
		log:  log,
		cfg:  con,
		cred: cred,
	}
}

// Connect opens the udp connection
func (c *Client) Connect() (err error) {
	c.Lock()
	defer c.Unlock()
	c.log.Info("creating new connection", zap.String("server", c.cfg.Addr.String()))
	c.con = connection.New(c.log)
	err = c.con.Connect(c.cfg.Addr)
	if err != nil {
		return err
	}
	c.log.Info("logging in")
	err = c.con.Login(c.cred.Password)
	if err != nil {
		return err
	}
	if !c.looping {
		c.looping = true
		c.init = true
		c.exit = false
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
		c.log.Info("reader exited", zap.Error(d))
		return d
	case d := <-wd:
		c.looping = false
		c.log.Info("writer exited", zap.Error(d))
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
					c.log.Info("failed to reconnect", zap.Error(err))
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
	c.log.Debug("handling response", zap.ByteString("response", response))
	trm, ex := c.con.GetTransmission(seq)
	if !ex {
		if len(response) == 0 {
			se := c.con.Sequence()
			if se == seq {
				c.con.AddPingback()
				c.log.Debug("received pingback", zap.Int64("count", c.con.KeepAlive()))
			}
		} else {
			c.log.Warn("no entry in cmdMap", zap.ByteString("response", response))
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
				c.log.Debug("writing to output", zap.ByteString("response", trm.Response))
				trm.WriteCloser.Write(trm.Response)
				if trm.WriteCloser != os.Stderr && trm.WriteCloser != os.Stdout {
					err := trm.WriteCloser.Close()
					if err != nil {
						c.log.Error("failed to write", zap.ByteString("response", trm.Response), zap.Error(err))
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
