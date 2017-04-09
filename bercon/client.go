package bercon

import (
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

//New creates a Client with given Config
func New(bec BeCfg) *Client {
	cfg := bec.GetConfig()
	if cfg.KeepAliveTimer == 0 {
		cfg.KeepAliveTimer = 10 //TODO: Evaluate default value
	}

	return &Client{
		addr:               cfg.Addr,
		password:           cfg.Password,
		keepAliveTimer:     cfg.KeepAliveTimer,
		keepAliveTolerance: cfg.KeepAliveTolerance,
		readBuffer:         make([]byte, 4096),
		reconnectTimeout:   25,
		cmdChan:            make(chan transmission),
		cmdMap:             make(map[byte]transmission),
	}
}

//Connect opens a new Connection to the Server
func (c *Client) Connect() (err error) {
	c.con, err = net.DialUDP("udp", nil, c.addr)
	if err != nil {
		c.con = nil
		return err
	}

	//Read Buffer
	buffer := make([]byte, 9)

	glog.V(2).Infoln("Sending Login Information")
	c.con.SetReadDeadline(time.Now().Add(time.Second * 2))
	c.con.Write(buildLoginPacket(c.password))
	n, err := c.con.Read(buffer)
	if err, ok := err.(net.Error); ok && err.Timeout() {
		c.con.Close()
		return ErrTimeout
	}
	if err != nil {
		c.con.Close()
		return err
	}

	response, err := verifyLogin(buffer[:n])
	if err != nil {
		c.con.Close()
		return err
	}
	if response == packetResponse.LoginFail {
		glog.Errorln("Non Login Packet Received:", response)
		c.con.Close()
		return ErrInvalidLogin
	}
	fmt.Println("Login successful")
	if !c.looping {
		c.looping = true
		c.init = true
		c.sequence.s = 0
		c.keepAliveCount = 0
		c.pingbackCount = 0
		c.cmdLock.Lock()
		c.cmdMap = make(map[byte]transmission)
		c.cmdLock.Unlock()

		go c.WatcherLoop()
	}
	return nil
}

//WatcherLoop is responsible for creating and keeping working connections
func (c *Client) WatcherLoop() {
	writerDisconnect := make(chan int)
	readerDisconnect := make(chan int)
	// Start Loops only if initial connection is up
	if c.init {
		go c.writerLoop(writerDisconnect, c.cmdChan)
		go c.readerLoop(readerDisconnect)
	}
	for {
		glog.V(10).Infoln("Looping in WatcherLoop")
		if !c.looping {
			if err := c.Reconnect(); err != nil {
				glog.V(2).Info(err)
				time.Sleep(time.Second * 3)
				continue
			}
			return
		}
		select {
		case d := <-readerDisconnect:
			c.looping = false
			glog.V(2).Infoln("Reader disconnected, waiting for Writer")
			_ = <-writerDisconnect
			glog.V(2).Infoln("Writer disconnected")
			glog.Warningf("Trying to recover from broken Connection (close msg: %v)", d)
			if err := c.Reconnect(); err == nil {
				return
			}
		case d := <-writerDisconnect:
			c.looping = false
			glog.V(2).Infoln("Writer disconnected, waiting for Reader")
			_ = <-readerDisconnect
			glog.V(2).Infoln("Reader disconnected")
			glog.Warningf("Trying to recover from broken Connection (close msg: %v)", d)
			if err := c.Reconnect(); err == nil {
				return
			}
			//TODO: Evaluate it this is required
			//default:
			//	continue
		}
	}
}

//Reconnect after loops exited or if not running
func (c *Client) Reconnect() error {
	//c.con.Close()
	var err error
	if err = c.Connect(); err == nil {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

//Disconnect the Client
func (c *Client) Disconnect() error {
	return nil
}

//SetChatWriter enables Chat Reading and sets Writer
func (c *Client) SetChatWriter(w io.Writer) {
	c.chatWriter.Lock()
	c.chatWriter.Writer = w
	c.chatWriter.Unlock()
}

//SetEventWriter enables Event Reading and sets Writer
func (c *Client) SetEventWriter(w io.Writer) {
	c.eventWriter.Lock()
	c.eventWriter.Writer = w
	c.eventWriter.Unlock()
}

//RunCommand adds given cmd to command queue
func (c *Client) RunCommand(cmd string, w io.WriteCloser) {
	c.cmdChan <- transmission{command: []byte(cmd), writeCloser: w}
}

func (c *Client) handleResponse(seq byte, response []byte, last bool) {
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
				trm.writeCloser.Write(trm.response)
				trm.writeCloser.Close()
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
