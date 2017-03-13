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
	//ErrConnectionNil .
	ErrConnectionNil = errors.New("Connection is nil")
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
	sequence    byte
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

	lastPacketTime time.Time
	sent           time.Time
	ready          bool
	con            *net.UDPConn
	readBuffer     []byte
	looping        bool

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

	writerQueue struct {
		sync.Mutex
		queue []transmission
	}

	cmdQueue struct {
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
		readBuffer:       make([]byte, 4096),
		reconnectTimeout: 25,
		looping:          false,
	}
}

//Connect opens a new Connection to the Server
func (c *Client) Connect() (err error) {
	if c.con != nil {
		c.con.Close()
	}
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
		glog.Errorln("Non Login Packet Received:", response)
		c.con.Close()
		return ErrInvalidLogin
	}
	if !c.looping {
		c.startLoops()
	}
	return nil
}

func (c *Client) startLoops() {
	c.looping = true
	c.waitGroup = sync.WaitGroup{}
	c.waitGroup.Add(2)
	c.lastPacketTime = time.Now()
	c.ready = true
	c.sequence.s = 0

	go c.writerLoop()
	go c.readerLoop()
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

//QueueCommand adds given cmd to command queue
func (c *Client) QueueCommand(cmd []byte, w io.WriteCloser) {
	c.writerQueue.Lock()
	c.writerQueue.queue = append(c.writerQueue.queue, transmission{command: cmd, writeCloser: w})
	c.writerQueue.Unlock()
}

func (c *Client) writerLoop() {
	defer c.waitGroup.Done()
	for {
		if c.con == nil {
			glog.Errorln(ErrConnectionNil)
			return
		}
		t := time.Now()

		//Connection KeepAlive
		c.lastPacket.Lock()
		if t.After(c.lastPacket.Add(time.Second * time.Duration(c.keepAliveTimer))) {
			if c.con != nil {
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err := c.con.Write(buildKeepAlivePacket(c.sequence.s))
				if err != nil {
					glog.Errorln(err)
				}
				c.lastPacket.Time = t
			}
		}
		c.lastPacket.Unlock()

		c.writerQueue.Lock()
		c.sequence.Lock()
		if len(c.writerQueue.queue) > 0 {
			trm := c.writerQueue.queue[0]
			if c.con != nil {
				c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
				trm.packet = buildCmdPacket(trm.command, c.sequence.s)
				glog.V(4).Infof("Sending Packet: %v - Command: %v - Sequence: %v", string(trm.packet), string(trm.command), c.sequence.s)
				_, err := c.con.Write(trm.packet)
				if err != nil {
					glog.Errorln(err)
				}
				c.lastPacket.Lock()
				c.lastPacket.Time = time.Now()
				c.lastPacket.Unlock()
				c.writerQueue.queue = c.writerQueue.queue[1:]
				trm.sequence = c.sequence.s
				c.cmdQueue.Lock()
				c.cmdQueue.queue = append(c.cmdQueue.queue, trm)
				c.cmdQueue.Unlock()
				c.sequence.s = c.sequence.s + 1
			} else {
				glog.Errorln(ErrConnectionNil)
				return
			}
		}
		c.sequence.Unlock()
		c.writerQueue.Unlock()
	}
}

func (c *Client) readerLoop() {
	defer c.waitGroup.Done()
	for {
		if c.con == nil {
			glog.Errorln(ErrConnectionNil)
			return
		}

		//c.con.SetReadDeadline(time.Now().Add(time.Millisecond)) Evaluate if Deadline is required
		n, err := c.con.Read(c.readBuffer)
		if err == nil {
			data := c.readBuffer[:n]
			if err := c.handlePacket(data); err != nil {
				glog.Errorln(err)
			}
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

	// Handle Packet Types
	if pType == packetType.ServerMessage {
		glog.V(4).Infof("ServerMessage Packet: %v - Sequence: %v", string(data), seq)
		c.handleServerMessage(append(data[3:], []byte("/n")...))
		if c.con != nil {
			c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
			_, err := c.con.Write(buildMsgAckPacket(seq))
			if err != nil {
				glog.Error(err)
				return err
			}
		}
		return nil
	}

	if pType != packetType.Command && pType != packetType.MultiCommand {
		glog.V(3).Infof("Packet: %v - PacketType: %v", string(packet), pType)
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
		if strings.Contains(string(data), "logged in") {
			glog.V(5).Infoln("Login Event: ", string(data))
			c.eventWriter.Writer.Write(data)
		} else {
			c.eventWriter.Writer.Write(data)
		}
	}
	c.eventWriter.Unlock()
}
