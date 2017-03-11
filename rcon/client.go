package rcon

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"strings"

	"github.com/golang/glog"
)

var (
	//ErrDisconnect .
	ErrDisconnect = errors.New("Connection lost")
	//ErrTimeout .
	ErrTimeout = errors.New("Connection timeout")
	//ErrInvalidLoginPacket .
	ErrInvalidLoginPacket = errors.New("Received invalid Login Packet")
	//ErrInvalidLogin .
	ErrInvalidLogin = errors.New("Login Invalid")
	//ErrInvalidChecksum .
	ErrInvalidChecksum = errors.New("Received invalid Packet Checksum")
	//ErrUnknownPacketType .
	ErrUnknownPacketType = errors.New("Received Unknown Packet Type")
)

//Config contains all data required by BE Connections
type Config struct {
	Addr           *net.UDPAddr
	Password       string
	KeepAliveTimer uint32
}

//GetConfig returns BeConfig
func (bec Config) GetConfig() Config {
	return bec
}

//BeCfg is the Interface providing Configs for the Client
type BeCfg interface {
	GetConfig() Config
}

type transmission struct {
	packet      []byte
	command     []byte
	response    []byte
	timestamp   time.Time
	writeCloser io.WriteCloser
}

//Client is the the Object Handling the Connection
type Client struct {

	//Config
	addr             *net.UDPAddr
	password         string
	keepAliveTimer   uint32
	reconnectTimeout float64

	lastPacketTime  time.Time
	currentSequence byte
	sent            time.Time
	ready           bool
	con             *net.UDPConn
	writeBuffer     []byte

	waitGroup sync.WaitGroup

	sequence struct {
		sync.Mutex
		s byte
	}

	conStatus struct {
		sync.Mutex
		bool
	}

	lastPacket struct {
		sync.Mutex
		time.Time
	}

	packetQueue struct {
		sync.Mutex
		queue []transmission
	}

	chatWriter struct {
		sync.Mutex
		io.Writer
	}

	eventWriter struct {
		sync.Mutex
		io.Writer
	}
}

//New creates a Client with given Config
func New(bec BeCfg) *Client {
	cfg := bec.GetConfig()
	if cfg.KeepAliveTimer == 0 {
		cfg.KeepAliveTimer = 10 //TODO: Evaluate default value
	}

	return &Client{
		addr:             cfg.Addr,
		password:         cfg.Password,
		keepAliveTimer:   cfg.KeepAliveTimer,
		writeBuffer:      make([]byte, 4096),
		reconnectTimeout: 25,
	}
}

//Connect opens a new Connection to the Server
func (c *Client) Connect() (err error) {
	c.con, err = net.DialUDP("udp", nil, c.addr)
	if err != nil {
		glog.Errorln("Connection failed")
		return err
	}

	//Read Buffer
	buffer := make([]byte, 9)

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
		c.con.Close()
		return ErrInvalidLogin
	}

	c.lastPacketTime = time.Now()
	c.ready = true
	c.currentSequence = 0

	c.waitGroup = sync.WaitGroup{}
	go c.loop()

	return nil
}

//Disconnect the Client
func (c *Client) Disconnect() error {
	c.con.Close()
	c.waitGroup.Wait()
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

func (c *Client) loop() {
	defer c.waitGroup.Done()
	for {
		if c.con == nil {
			glog.Errorln("Connection is nil")
			return
		}
		t := time.Now()
		if t.Sub(c.lastPacketTime).Seconds() > c.reconnectTimeout {
			c.lastPacketTime = t
			c.Connect()
		}

		c.lastPacket.Lock()
		if t.After(c.lastPacket.Add(time.Second * time.Duration(c.keepAliveTimer))) {
			//TODO: Check KeepAlivePacket
			if c.con != nil {
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err := c.con.Write(buildKeepAlivePacket(c.sequence.s))
				if err != nil {
					glog.Error(err)
				}
			}
			c.lastPacket.Unlock()
		} else {
			c.lastPacket.Unlock()
		}

		c.con.SetReadDeadline(time.Now().Add(time.Millisecond))
		n, err := c.con.Read(c.writeBuffer)
		if err == nil {
			data := c.writeBuffer[:n]
			if err := c.handlePacket(data); err != nil {
				glog.Errorln(err)
			}
		}

		if time.Now().After(c.sent.Add(time.Second*2)) || c.ready {
			c.packetQueue.Lock()
			if len(c.packetQueue.queue) == 0 {
				c.packetQueue.Unlock()
				continue
			}
			trm := c.packetQueue.queue[0]
			c.packetQueue.queue[0].response = nil
			packet := buildCmdPacket(trm.command, c.currentSequence)
			c.con.SetWriteDeadline(time.Now().Add(time.Second * 2))
			c.con.Write(packet)
			c.ready = false
			c.sent = time.Now()
			c.packetQueue.Unlock()
		}
	}
}

func verifyLogin(packet []byte) (byte, error) {
	if len(packet) != 9 {
		return 0, ErrInvalidLoginPacket
	}
	if match, err := verifyChecksumMatch(packet); match == false || err != nil {
		return 0, ErrInvalidChecksum
	}

	return packet[8], nil
}

func (c *Client) handlePacket(packet []byte) error {
	seq, data, pType, err := verifyPacket(packet)
	if err != nil {
		glog.Errorln(err)
		return err
	}

	if pType == packetType.ServerMessage {
		c.handleServerMessage(append(packet[3:], []byte("/n")...))
		if c.con != nil {
			c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
			_, err := c.con.Write(buildMsgAckPacket(seq))
			if err != nil {
				glog.Error(err)
				return err
			}
		}
	}

	if pType != packetType.Command && pType != 0x00 {
		return ErrUnknownPacketType
	}

	packetCount, currentPacket, isMultiPacket := checkMultiPacketResponse(data)
	if !isMultiPacket {
		c.handleResponse(seq, data[3:], true)
		return nil
	}

	if currentPacket+1 < packetCount {
		c.handleResponse(seq, data[6:], false)
	} else {
		c.handleResponse(seq, data[6:], true)
	}
	return nil
}

func (c *Client) handleResponse(seq byte, response []byte, last bool) {
	c.packetQueue.Lock()

	if len(c.packetQueue.queue) == 0 {
		glog.Warningln("Queue Empty which is unexpected")
		return
	}

	trm := c.packetQueue.queue[0]
	lineEnd := []byte("\n")
	if !last {
		lineEnd = []byte{}
	}
	trm.response = append(trm.response, response...)
	trm.response = append(trm.response, lineEnd...)

	c.packetQueue.queue[0] = trm
	if last {
		if trm.writeCloser != nil {
			trm.writeCloser.Write(trm.response)
			trm.writeCloser.Close()
		}
		c.packetQueue.queue = c.packetQueue.queue[1:]
		c.currentSequence = seq + 1
		c.ready = true
	}
	c.packetQueue.Unlock()
}

func (c *Client) handleServerMessage(data []byte) {
	var ChatPatterns = []string{
		/*"RCon admin", <- Kicked out to handle as event? */
		"(Group)",
		"(Vehicle)",
		"(Unknown)",
	}
	for _, v := range ChatPatterns {
		if strings.HasPrefix(string(data), v) {
			/*if v == "RCon admin" {
				if strings.HasSuffix(string(data), "logged in\n") {
					//TODO: Handle Login?
					break
				}
			}*/
			c.chatWriter.Lock()
			if c.chatWriter.Writer != nil {
				c.chatWriter.Write(data)
			}
			c.chatWriter.Unlock()
			return
		}
	}
	c.eventWriter.Lock()
	if c.eventWriter.Writer != nil {
		c.eventWriter.Write(data)
	}
	c.eventWriter.Unlock()
}
