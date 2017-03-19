package rcon

import (
	"time"

	"github.com/golang/glog"
)

func (c *Client) writerLoop(disc chan int, cmd chan transmission) {
	defer func(disc chan int) { disc <- 1 }(disc)
	for {
		if !c.looping {
			glog.V(3).Infoln("WriterLoop ended by watcher. Exiting.")
			return
		}
		if c.con == nil {
			glog.Errorln(ErrConnectionNil)
			return
		}

		t := time.Now()
		timeout := time.After(time.Second * time.Duration(c.keepAliveTimer))

		select {
		case trm := <-cmd:
			glog.V(4).Infoln("Preparing Command: ", trm)
			c.writeCommand(trm)
		case <-timeout:
			if c.con != nil {
				glog.Infof("Sending Keepalive")
				c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
				_, err := c.con.Write(buildKeepAlivePacket(c.sequence.s))
				if err != nil {
					glog.Errorln(err)
					return
				}
				c.lastPacket.Lock()
				c.lastPacket.Time = t
				c.lastPacket.Unlock()
			}
		}
	}
}

func (c *Client) writeCommand(trm transmission) error {
	c.sequence.Lock()
	if c.con != nil {
		c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
		trm.packet = buildCmdPacket(trm.command, c.sequence.s)
		glog.V(4).Infof("Sending Packet: %v - Command: %v - Sequence: %v", string(trm.packet), string(trm.command), c.sequence.s)
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
