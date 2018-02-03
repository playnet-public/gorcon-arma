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

type connect func(q chan error) error
type disconnect func() error
type exec func(cmd []byte, resp io.WriteCloser) error
type attachEvents func(w io.Writer) error
type attachChat func(w io.Writer) error

//Client implements an abstract rcon client object
type Client struct {
	Connect      connect
	Disconnect   disconnect
	Exec         exec
	AttachEvents attachEvents
	AttachChat   attachChat
}

//NewClient returns an abstract rcon client
func NewClient(
	connect connect,
	disconnect disconnect,
	exec exec,
	attachEvents attachEvents,
	attachChat attachChat,
) *Client {
	c := new(Client)
	c.Connect = connect
	c.Disconnect = disconnect
	c.Exec = exec
	c.AttachEvents = attachEvents
	c.AttachChat = attachChat
	return c
}
