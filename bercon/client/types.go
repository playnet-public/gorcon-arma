package client

import (
	"io"
	"net"
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
)

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
	cfg  rcon.Connection
	cred rcon.Credentials
	con  *net.UDPConn

	init       bool
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

//PacketType contains all possible types a packet could have
var PacketType = struct {
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

//PacketResponse contains all types a server could respond with
var PacketResponse = struct {
	LoginOk      byte
	LoginFail    byte
	MultiCommand byte
}{
	LoginOk:      0x01,
	LoginFail:    0x00,
	MultiCommand: 0x00,
}

//ResponseType gets the PacketResponse from a packet
func ResponseType(data []byte) (byte, error) {
	if len(data) < 8 {
		raven.CaptureError(common.ErrInvalidSize, nil)
		return 0, common.ErrInvalidSize
	}
	return data[7], nil
}

//GetSequence extracts the seq number from a packet
func GetSequence(data []byte) (byte, error) {
	if len(data) < 9 {
		raven.CaptureError(common.ErrInvalidSizeNoSequence, nil)
		return 0, common.ErrInvalidSizeNoSequence
	}
	return data[8], nil
}
