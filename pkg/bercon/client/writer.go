package client

import (
	"time"

	"fmt"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/common"
)

func (c *Client) writerLoop(ret chan error, cmd chan protocol.Transmission) {
	var err error
	c.RLock()
	defer c.RUnlock()
	defer func() { ret <- err }()
	for {
		glog.V(10).Infoln("Looping in writerLoop")
		if !c.looping {
			glog.V(4).Infoln("WriterLoop ended externally. Exiting.")
			return
		}
		if c.con.UDPConn == nil {
			glog.Errorln(common.ErrConnectionNil)
			raven.CaptureError(common.ErrConnectionNil, map[string]string{"app": "rcon", "module": "writer"})
			err = common.ErrConnectionNil
			return
		}

		timeout := time.After(time.Second * time.Duration(c.cfg.KeepAliveTimer))

		select {
		case trm := <-cmd:
			glog.V(4).Infoln("Preparing Command: ", trm)
			err = c.writeCommand(trm)
			if err != nil {
				glog.Error(err)
				return
			}
		case <-timeout:
			if c.con.UDPConn != nil {
				glog.V(3).Infof("Sending Keepalive")
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err = c.con.Write(protocol.BuildKeepAlivePacket(c.con.Sequence()))
				if err != nil {
					glog.Errorln(err)
					return
				}
				keepAliveCount := c.con.KeepAlive()
				pingbackCount := c.con.Pingback()
				if diff := keepAliveCount - pingbackCount; diff > c.cfg.KeepAliveTolerance || diff < c.cfg.KeepAliveTolerance*-1 {
					err = fmt.Errorf("KeepAlive Packets are out of sync by %v", diff)
					glog.Errorln(err)
					raven.CaptureError(err, map[string]string{"app": "rcon", "module": "writer"})
					return
				}
				// Experimental change to check if growing count is causing performance leak
				//TODO: Evaluate if this is still required
				if keepAliveCount > 20 {
					c.con.ResetPingback()
					c.con.ResetKeepAlive()
				}
			}
		}
	}
}

func (c *Client) writeCommand(trm protocol.Transmission) error {
	if c.con.UDPConn != nil {
		c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
		trm.Packet = protocol.BuildCmdPacket(trm.Command, c.con.Sequence())
		glog.V(3).Infof("Sending Packet: %v - Command: %v - Sequence: %v", string(trm.Packet), string(trm.Command), c.con.Sequence())
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
