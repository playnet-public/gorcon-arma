package client

import (
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"go.uber.org/zap"
)

func (c *Client) writerLoop(ret chan error, cmd chan protocol.Transmission) {
	var err error
	c.RLock()
	defer c.RUnlock()
	defer func() { ret <- err }()
	for c.con == nil {
		c.log.Debug("looping writer")

		select {
		case trm := <-cmd:
			c.log.Debug("preparing command", zap.ByteString("command", trm.Command))
			err = c.writeCommand(trm)
			if err != nil {
				c.log.Error("write command error", zap.ByteString("command", trm.Command), zap.Error(err))
				return
			}
		case <-time.After(time.Second * time.Duration(c.cfg.KeepAliveTimer)):
			err = c.KeepAlive(c.con.Sequence())
			if err != nil {
				c.log.Error("write keepalive error", zap.Uint32("seq", c.con.Sequence()), zap.Error(err))
				return
			}
		}
	}
	c.log.Info("loop exited", zap.String("loop", "writer"))
}

func (c *Client) writeCommand(trm protocol.Transmission) error {
	if c.con.UDPConn != nil {
		c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
		trm.Packet = protocol.BuildCmdPacket(trm.Command, c.con.Sequence())
		c.log.Debug("sending packet", zap.Uint32("seq", c.con.Sequence()), zap.ByteString("cmd", trm.Command), zap.ByteString("packet", trm.Packet))
		_, err := c.con.Write(trm.Packet)
		if err != nil {
			return err
		}
		trm.Sequence = c.con.Sequence()
		c.con.SetTransmission(c.con.Sequence(), trm)
		c.con.AddSequence()
	}
	return nil
}
