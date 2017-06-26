package client

import (
	"sync/atomic"
	"time"

	"fmt"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/bercon/common"
)

func (c *Client) writerLoop(disc chan int, cmd chan transmission) {
	defer func(disc chan int) { disc <- 1 }(disc)
	for {
		glog.V(10).Infoln("Looping in writerLoop")
		if !c.looping {
			glog.V(4).Infoln("WriterLoop ended by watcher. Exiting.")
			return
		}
		if c.con == nil {
			glog.Errorln(common.ErrConnectionNil)
			raven.CaptureError(common.ErrConnectionNil, map[string]string{"app": "rcon", "module": "writer"})
			return
		}

		timeout := time.After(time.Second * time.Duration(c.cfg.KeepAliveTimer))

		select {
		case trm := <-cmd:
			glog.V(4).Infoln("Preparing Command: ", trm)
			err := c.writeCommand(trm)
			if err != nil {
				glog.Error(err)
				return
			}
		case <-timeout:
			if c.con != nil {
				glog.V(3).Infof("Sending Keepalive")
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err := c.con.Write(common.BuildKeepAlivePacket(c.sequence.s))
				if err != nil {
					glog.Errorln(err)
					return
				}
				keepAliveCount := atomic.AddInt64(&c.keepAliveCount, 1)
				pinbackCount := atomic.LoadInt64(&c.pingbackCount)
				if diff := keepAliveCount - pinbackCount; diff > c.cfg.KeepAliveTolerance || diff < c.cfg.KeepAliveTolerance*-1 {
					glog.Errorf("KeepAlive Packets are out of sync by %v", diff)
					raven.CaptureError(fmt.Errorf("KeepAlive Packets are out of sync by %v", diff), map[string]string{"app": "rcon", "module": "writer"})
					return
				}
				// Experimental change to check if growing count is causing performance leak
				if keepAliveCount > 20 {
					atomic.SwapInt64(&c.keepAliveCount, 0)
					atomic.SwapInt64(&c.pingbackCount, 0)
				}
			}
		}
	}
}

func (c *Client) writeCommand(trm transmission) error {
	c.sequence.Lock()
	if c.con != nil {
		c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
		trm.packet = common.BuildCmdPacket(trm.command, c.sequence.s)
		glog.V(3).Infof("Sending Packet: %v - Command: %v - Sequence: %v", string(trm.packet), string(trm.command), c.sequence.s)
		_, err := c.con.Write(trm.packet)
		if err != nil {
			c.sequence.Unlock()
			return err
		}
		trm.sequence = c.sequence.s
		c.cmdLock.Lock()
		c.cmdMap[c.sequence.s] = trm
		c.cmdLock.Unlock()
		c.sequence.s = c.sequence.s + 1
	}
	c.sequence.Unlock()
	return nil
}
