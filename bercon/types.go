package bercon

import (
	"io"
	"net"
	"sync"
	"time"
)

//Config contains all data required by BE Connections
type Config struct {
	Addr               *net.UDPAddr
	Password           string
	KeepAliveTimer     int
	KeepAliveTolerance int64
}

//BeCfg is the Interface providing Configs for the Client
type BeCfg interface {
	GetConfig() Config
}

//GetConfig returns BeConfig
func (bec Config) GetConfig() Config {
	return bec
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
	addr               *net.UDPAddr
	password           string
	keepAliveTimer     int
	keepAliveTolerance int64
	reconnectTimeout   int

	init       bool
	con        *net.UDPConn
	readBuffer []byte
	cmdChan    chan transmission
	looping    bool

	sequence struct {
		sync.RWMutex
		s byte
	}

	cmdMap  map[byte]transmission
	cmdLock sync.RWMutex

	keepAliveCount int64
	pingbackCount  int64

	chatWriter struct {
		sync.Mutex
		io.Writer
	}

	eventWriter struct {
		sync.Mutex
		io.Writer
	}
}

var packetType = struct {
	Login         byte
	Command       byte
	MultiCommand  byte
	ServerMessage byte
}{
	Login:         0x00,
	Command:       0x01,
	MultiCommand:  0x00,
	ServerMessage: 0x02,
}

var packetResponse = struct {
	LoginOk      byte
	LoginFail    byte
	MultiCommand byte
}{
	LoginOk:      0x01,
	LoginFail:    0x00,
	MultiCommand: 0x00,
}

func responseType(data []byte) (byte, error) {
	if len(data) < 8 {
		return 0, ErrInvalidSize
	}
	return data[7], nil
}

func getSequence(data []byte) (byte, error) {
	if len(data) < 9 {
		return 0, ErrInvalidSizeNoSequence
	}
	return data[8], nil
}
