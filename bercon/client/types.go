package client

import (
	"io"
	"net"
	"sync"
	"time"
)

//Credentials implements a common struct for storing auth information
type Credentials struct {
	Username string
	Password string
}

//Connection implements a common struct for storing connection information
type Connection struct {
	Addr               *net.UDPAddr
	KeepAliveTimer     int
	KeepAliveTolerance int64
	ReconnectTimeout   int
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
	cfg  Connection
	cred Credentials
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
