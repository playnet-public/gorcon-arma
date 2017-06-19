package rcon

import (
	"io"
	"net"
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

//Client implements an abstract rcon client object
type Client struct {
	Connect    connect
	Disconnect disconnect
	Exec       exec
}

type connect func() error
type disconnect func() error
type exec func(cmd []byte, resp io.WriteCloser) error

//NewClient returns an abstract rcon client
func NewClient(
	connect connect,
	disconnect disconnect,
	exec exec,
) *Client {
	c := new(Client)
	c.Connect = connect
	c.Disconnect = disconnect
	c.Exec = exec
	return c
}
