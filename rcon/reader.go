package rcon

import (
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
)

func (c *Client) readerLoop(disc chan int) {
	defer func(disc chan int) { disc <- 2 }(disc)
	for {
		if !c.looping {
			glog.V(4).Infoln("ReaderLoop ended by watcher. Exiting.")
			return
		}
		if c.con == nil {
			glog.Errorln(ErrConnectionNil)
			return
		}

		c.con.SetReadDeadline(time.Now().Add(time.Second * 2)) //Evaluate if Deadline is required
		n, err := c.con.Read(c.readBuffer)
		if err == nil {
			data := c.readBuffer[:n]
			glog.V(5).Infof("Received Data: %v", data)
			if herr := c.handlePacket(data); herr != nil {
				glog.Errorln(err)
			}
			//TODO: Evaluate if parallel aproach is better
			//go c.handlePacket(data)
		}
		if err != nil {
			if err, _ := err.(net.Error); err.Timeout() {
				glog.V(5).Infoln(err)
				continue
			} else {
				glog.Error(err)
				return
			}
		}

	}
}

func (c *Client) handlePacket(packet []byte) error {
	seq, data, pType, err := verifyPacket(packet)
	if err != nil {
		glog.Errorln(err)
		return err
	}

	// Handle Packet Types
	if pType == packetType.ServerMessage {
		glog.V(3).Infof("ServerMessage Packet: %v - Sequence: %v", string(data), seq)
		c.handleServerMessage(append(data[3:], []byte("/n")...))
		if c.con != nil {
			c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
			_, err := c.con.Write(buildMsgAckPacket(seq))
			if err != nil {
				glog.Error(err)
				return err
			}
		}
		return nil
	}

	if pType != packetType.Command && pType != packetType.MultiCommand {
		glog.V(2).Infof("Packet: %v - PacketType: %v", string(packet), pType)
		return ErrUnknownPacketType
	}

	packetCount, currentPacket, isMultiPacket := checkMultiPacketResponse(data)
	glog.V(3).Infof("Packet: %v - Sequence: %v - IsMulti: %v", string(data), seq, isMultiPacket)
	if !isMultiPacket {
		c.handleResponse(seq, data[3:], true)
		return nil
	}

	if currentPacket+1 < packetCount {
		c.handleResponse(seq, data[6:], false)
	} else {
		c.handleResponse(seq, data[6:], true)
	}

	return nil
}

func (c *Client) handleServerMessage(data []byte) {
	var ChatPatterns = []string{
		/*"RCon admin", <- Kicked out to handle as event? */
		"(Group)",
		"(Vehicle)",
		"(Unknown)",
	}
	for _, v := range ChatPatterns {
		if strings.HasPrefix(string(data), v) {
			c.chatWriter.Lock()
			if c.chatWriter.Writer != nil {
				c.chatWriter.Write(data)
			}
			c.chatWriter.Unlock()
			return
		}
	}
	c.eventWriter.Lock()
	if c.eventWriter.Writer != nil {
		if strings.Contains(string(data), "logged in") {
			glog.V(2).Infoln("Login Event: ", string(data))
			c.eventWriter.Writer.Write(data)
		} else {
			c.eventWriter.Writer.Write(data)
		}
	}
	c.eventWriter.Unlock()
}
