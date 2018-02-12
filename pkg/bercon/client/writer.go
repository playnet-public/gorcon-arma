package client

import (
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"go.uber.org/zap"
)

func (c *Client) writerLoop(ret chan error, cmd chan protocol.Transmission) {
	var err error
	c.RLock()
	defer c.RUnlock()
	defer func() { ret <- err }()
	for {
		c.log.Debug("looping writer")
		if !c.looping {
			c.log.Info("loop exited", zap.String("loop", "writer"))
			return
		}
		if c.con.UDPConn == nil {
			c.log.Error("invalid connection", zap.Error(common.ErrConnectionNil))
			err = common.ErrConnectionNil
			return
		}

		timeout := time.After(time.Second * time.Duration(c.cfg.KeepAliveTimer))

		select {
		case trm := <-cmd:
			c.log.Debug("preparing command", zap.ByteString("command", trm.Command))
			err = c.writeCommand(trm)
			if err != nil {
				c.log.Error("write command error", zap.ByteString("command", trm.Command), zap.Error(err))
				return
			}
		case <-timeout:
			if c.con.UDPConn != nil {
				c.log.Debug("sending keepalive", zap.Int64("count", c.con.KeepAlive()))
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err = c.con.Write(protocol.BuildKeepAlivePacket(c.con.Sequence()))
				if err != nil {
					c.log.Error("send keepalive error", zap.Error(err))
					return
				}
				c.con.AddKeepAlive()
				keepAliveCount := c.con.KeepAlive()
				pingbackCount := c.con.Pingback()
				if diff := keepAliveCount - pingbackCount; diff > c.cfg.KeepAliveTolerance || diff < c.cfg.KeepAliveTolerance*-1 {
					err = common.ErrKeepAliveAsync
					c.log.Error("keepalive out of sync", zap.Int64("count", diff))
					return
				}
				// Experimental change to check if growing count is causing performance leak
				//TODO: Evaluate if this is still required
				/*if keepAliveCount > 20 {
					c.con.ResetPingback()
					c.con.ResetKeepAlive()
				}*/
			}
		}
	}
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
