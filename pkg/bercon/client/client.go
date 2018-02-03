package client

import (
	"io"
	"net"
	"sync/atomic"
	"time"

	"os"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
)

//New creates a Client with given Config
func New(con rcon.Connection, cred rcon.Credentials) *Client {
	if con.KeepAliveTimer == 0 {
		con.KeepAliveTimer = 10 //TODO: Evaluate default value
	}

	return &Client{
		cfg:        con,
		cred:       cred,
		readBuffer: make([]byte, 4096),
		cmdChan:    make(chan transmission),
		cmdMap:     make(map[byte]transmission),
	}
}

//Connect opens a new Connection to the Server
func (c *Client) Connect(q chan error) error {
	var err error
	c.con, err = net.DialUDP("udp", nil, c.cfg.Addr)
	if err != nil {
		c.con = nil
		return err
	}

	//Read Buffer
	buffer := make([]byte, 9)

	glog.V(2).Infoln("Sending Login Information")
	c.con.SetReadDeadline(time.Now().Add(time.Second * 2))
	c.con.Write(BuildLoginPacket(c.cred.Password))
	n, err := c.con.Read(buffer)
	if err, ok := err.(net.Error); ok && err.Timeout() {
		c.con.Close()
		return common.ErrTimeout
	}
	if err != nil {
		c.con.Close()
		return err
	}

	response, err := VerifyLogin(buffer[:n])
	if err != nil {
		c.con.Close()
		return err
	}
	if response == PacketResponse.LoginFail {
		glog.Errorln("Non Login Packet Received:", response)
		c.con.Close()
		return common.ErrInvalidLogin
	}
	glog.Infoln("Login successful")
	if !c.looping {
		c.looping = true
		c.init = true
		c.exit = false
		c.sequence.s = 0
		c.keepAliveCount = 0
		c.pingbackCount = 0
		c.cmdLock.Lock()
		c.cmdMap = make(map[byte]transmission)
		c.cmdLock.Unlock()
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
				if err := c.Connect(q); err != nil {
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
	c.cmdChan <- transmission{command: cmd, writeCloser: resp}
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

func (c *Client) handleResponse(seq byte, response []byte, last bool) {
	glog.V(6).Infoln("Handling Response:", response, string(response))
	c.cmdLock.RLock()
	trm, ex := c.cmdMap[seq]
	c.cmdLock.RUnlock()
	if !ex {
		if len(response) == 0 {
			c.sequence.Lock()
			se := c.sequence.s
			c.sequence.Unlock()
			if se == seq {
				glog.V(3).Infoln("Received KeepAlive Pingback")
				atomic.AddInt64(&c.pingbackCount, 1)
			}
		} else {
			glog.Warningf("No Entry in cmdMap for: %v - (%v)", string(response), response)
		}
	} else {
		trail := []byte("\n")
		if !last {
			trail = []byte{}
		}
		trm.response = append(trm.response, response...)
		trm.response = append(trm.response, trail...)
		if last {
			if trm.writeCloser != nil {
				glog.V(4).Infoln("Writing", string(trm.response), "to output")
				trm.writeCloser.Write(trm.response)
				if trm.writeCloser != os.Stderr && trm.writeCloser != os.Stdout {
					err := trm.writeCloser.Close()
					if err != nil {
						glog.Errorln(err)
						raven.CaptureError(err, map[string]string{"app": "rcon", "module": "client"})
					}
				}
			}

			//TODO: Evaluate if this is required
			go func(c *Client, seq byte) {
				c.cmdLock.Lock()
				delete(c.cmdMap, seq)
				c.cmdLock.Unlock()
			}(c, seq)
		}
	}
}
