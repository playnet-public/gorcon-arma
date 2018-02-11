package client

import (
	"io"
	"sync"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/connection"
	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
)

//Client is the the Object Handling the Connection
type Client struct {
	cfg  rcon.Connection
	cred rcon.Credentials
	con  *connection.Conn
	sync.RWMutex

	init       bool
	exit       bool
	readBuffer []byte
	cmdChan    chan protocol.Transmission
	looping    bool

	chatWriter struct {
		sync.Mutex
		io.Writer
	}

	eventWriter struct {
		sync.Mutex
		io.Writer
	}
}
