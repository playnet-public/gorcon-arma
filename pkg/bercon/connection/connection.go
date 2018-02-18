package connection

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/playnet-public/libs/log"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"go.uber.org/zap"
)

// Conn to BattlEye
type Conn struct {
	log *log.Logger
	*net.UDPConn
	readBuffer     []byte
	seq            uint32
	keepAliveCount int64
	pingbackCount  int64

	cmd struct {
		m map[uint32]protocol.Transmission
		sync.RWMutex
	}
}

// New Connection to BattlEye
func New(log *log.Logger) *Conn {
	c := &Conn{
		log:        log,
		readBuffer: make([]byte, 4096),
	}
	atomic.StoreUint32(&c.seq, 0)
	atomic.StoreInt64(&c.keepAliveCount, 0)
	atomic.StoreInt64(&c.pingbackCount, 0)
	c.cmd.m = make(map[uint32]protocol.Transmission)
	return c
}

// Connect opens the udp connection
func (c *Conn) Connect(addr *net.UDPAddr) (err error) {
	c.UDPConn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		c.UDPConn = nil
		return err
	}
	c.SetReadDeadline(time.Now().Add(time.Second * 2)) //Evaluate if Deadline is required
	c.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))

	return nil
}

// Login sends auth info to BE
func (c *Conn) Login(pass string) (err error) {
	buffer := make([]byte, 9)

	c.Write(protocol.BuildLoginPacket(pass))

	n, err := c.UDPConn.Read(buffer)
	if err, ok := err.(net.Error); ok && err.Timeout() {
		c.Close()
		return common.ErrTimeout
	}
	if err != nil {
		c.log.Error("login failed", zap.Error(err))
		c.Close()
		return err
	}

	response, err := protocol.VerifyLogin(buffer[:n])
	if err != nil {
		c.log.Error("login failed", zap.ByteString("response", []byte{response}), zap.Error(err))
		c.Close()
		return err
	}
	if response == protocol.PacketResponse.LoginFail {
		c.log.Error("login failed", zap.ByteString("response", []byte{response}), zap.Error(common.ErrInvalidLogin))
		c.Close()
		return common.ErrInvalidLogin
	}
	c.log.Info("login successful")
	return nil
}

// Read from udp and return data or error
func (c *Conn) Read() ([]byte, error) {
	if c.UDPConn == nil {
		c.log.Error("connection nil", zap.Error(common.ErrConnectionNil))
		return nil, common.ErrConnectionNil
	}
	n, err := c.UDPConn.Read(c.readBuffer)
	if err != nil {
		return []byte(""), err
	}
	return c.readBuffer[:n], nil
}

// WriteAck sends a new AckPacket via UDP
func (c *Conn) WriteAck(seq uint32) (err error) {
	if c.UDPConn == nil {
		c.log.Error("connection nil", zap.Error(common.ErrConnectionNil))
		return common.ErrConnectionNil
	}
	_, err = c.Write(protocol.BuildMsgAckPacket(seq))
	if err != nil {
		c.log.Debug("write ack error", zap.Uint32("seq", seq), zap.Error(err))
	}
	return err
}

// WriteKeepAlive sends a new KeepAlivePacket via UDP
func (c *Conn) WriteKeepAlive(seq uint32) (err error) {
	if c.UDPConn == nil {
		c.log.Error("connection nil", zap.Error(common.ErrConnectionNil))
		return common.ErrConnectionNil
	}
	_, err = c.Write(protocol.BuildKeepAlivePacket(seq))
	if err != nil {
		c.log.Debug("write keepalive error", zap.Uint32("seq", seq), zap.Error(err))
	}
	return err
}

// GetTransmission from cmd.map
func (c *Conn) GetTransmission(seq uint32) (protocol.Transmission, bool) {
	c.cmd.RLock()
	defer c.cmd.RUnlock()
	trm, ok := c.cmd.m[seq]
	return trm, ok
}

// DeleteTransmission from cmd.map
func (c *Conn) DeleteTransmission(seq uint32) {
	c.cmd.Lock()
	defer c.cmd.Unlock()
	delete(c.cmd.m, seq)
}

// SetTransmission from cmd.map
func (c *Conn) SetTransmission(seq uint32, trm protocol.Transmission) {
	c.cmd.Lock()
	defer c.cmd.Unlock()
	c.cmd.m[seq] = trm
}

// Sequence gets the current sequence using atomic
func (c *Conn) Sequence() uint32 {
	return atomic.LoadUint32(&c.seq)
}

// AddSequence increments the sequence
func (c *Conn) AddSequence() {
	atomic.AddUint32(&c.seq, 1)
}

// Pingback gets the current pingbackCount using atomic
func (c *Conn) Pingback() int64 {
	return atomic.LoadInt64(&c.pingbackCount)
}

// AddPingback increments the pingbackCount
func (c *Conn) AddPingback() {
	atomic.AddInt64(&c.pingbackCount, 1)
}

// ResetPingback increments the pingbackCount
func (c *Conn) ResetPingback() {
	atomic.SwapInt64(&c.pingbackCount, 0)
}

// KeepAlive gets the current keepAliveCount using atomic
func (c *Conn) KeepAlive() int64 {
	return atomic.LoadInt64(&c.keepAliveCount)
}

// AddKeepAlive increments the keepAliveCount
func (c *Conn) AddKeepAlive() {
	atomic.AddInt64(&c.keepAliveCount, 1)
}

// ResetKeepAlive increments the keepAliveCount
func (c *Conn) ResetKeepAlive() {
	atomic.SwapInt64(&c.keepAliveCount, 0)
}
