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
			glog.V(3).Infoln("Preparing Command: ", trm)
			trm.writeCloser.Write([]byte("Command Result Placeholder"))
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

		/*
			c.writerQueue.Lock()
			c.sequence.Lock()
			if len(c.writerQueue.queue) > 0 {
				trm := c.writerQueue.queue[0]
				if c.con != nil {
					c.con.SetWriteDeadline(time.Now().Add(time.Second * 2)) //TODO: Evaluate Deadlines
					trm.packet = buildCmdPacket(trm.command, c.sequence.s)
					glog.V(4).Infof("Sending Packet: %v - Command: %v - Sequence: %v", string(trm.packet), string(trm.command), c.sequence.s)
					_, err := c.con.Write(trm.packet)
					if err != nil {
						glog.Errorln(err)
						c.sequence.Unlock()
						c.writerQueue.Unlock()
						return
					}
					c.lastPacket.Lock()
					c.lastPacket.Time = time.Now()
					c.lastPacket.Unlock()
					c.writerQueue.queue = c.writerQueue.queue[1:]
					trm.sequence = c.sequence.s
					c.cmdQueue.Lock()
					//c.cmdQueue.queue = append(c.cmdQueue.queue, trm) Try not to append but insert with sequence as id
					//if c.cmdQueue.queue[trm.sequence].packet != nil {
					//	glog.Warningf("Overwriting exisiting transmission in cmdQueue: %v - with: %v", c.cmdQueue.queue[trm.sequence], trm)
					//}
					if len(c.cmdQueue.queue) > 0 {
						c.cmdQueue.queue[trm.sequence] = trm
					}
					c.cmdQueue.Unlock()
					c.sequence.s = c.sequence.s + 1
				} else {
					glog.Errorln(ErrConnectionNil)
					c.sequence.Unlock()
					c.writerQueue.Unlock()
					return
				}
			}
			c.sequence.Unlock()
			c.writerQueue.Unlock()
		*/
	}
}
