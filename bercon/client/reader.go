package client

import (
	"net"
	"strings"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/common"
)

func (c *Client) readerLoop(ret chan error) {
	var err error
	defer func() { ret <- err }()
	for {
		glog.V(10).Infoln("Looping in readerLoop")
		if !c.looping {
			glog.V(4).Infoln("ReaderLoop ended externally. Exiting.")
			//TODO: Should we place some error return here?
			return
		}
		if c.con == nil {
			glog.Errorln(common.ErrConnectionNil)
			err = common.ErrConnectionNil
			return
		}

		c.con.SetReadDeadline(time.Now().Add(time.Second * 2)) //Evaluate if Deadline is required
		n, err := c.con.Read(c.readBuffer)
		if err == nil {
			data := c.readBuffer[:n]
			glog.V(5).Infof("Received Data: %v", data, "-", string(data))
			if herr := c.handlePacket(data); herr != nil {
				raven.CaptureError(err, map[string]string{"app": "rcon", "module": "reader"})
				glog.Errorln(herr)
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
				raven.CaptureErrorAndWait(err, map[string]string{"app": "rcon", "module": "reader"})
				return
			}
		}

	}
}

func (c *Client) handlePacket(packet []byte) error {
	seq, data, pType, err := VerifyPacket(packet)
	if err != nil {
		glog.Errorln(err)
		return err
	}

	// Handle Packet Types
	if pType == PacketType.ServerMessage {
		glog.V(3).Infof("ServerMessage Packet: %v - Sequence: %v", string(data), seq)
		c.handleServerMessage(append(data[3:], []byte("\n")...))
		if c.con != nil {
			c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
			_, err := c.con.Write(BuildMsgAckPacket(seq))
			if err != nil {
				glog.Error(err)
				return err
			}
		}
		return nil
	}

	if pType != PacketType.Command && pType != PacketType.MultiCommand {
		glog.V(2).Infof("Packet: %v - PacketType: %v", string(packet), pType)
		raven.CaptureError(common.ErrUnknownPacketType, map[string]string{"packetType": string(pType), "app": "rcon", "module": "reader"})
		return common.ErrUnknownPacketType
	}

	packetCount, currentPacket, isMultiPacket := CheckMultiPacketResponse(data)
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
		"(Group)",
		"(Vehicle)",
		"(Unknown)",
	}
	for _, v := range ChatPatterns {
		glog.V(10).Infoln("Looping in handleServerMessage")
		if strings.HasPrefix(string(data), v) {
			if c.chatWriter.Writer != nil {
				c.chatWriter.Lock()
				_, err := c.chatWriter.Write(data)
				if err != nil {
					raven.CaptureError(err, map[string]string{"app": "rcon", "module": "reader"})
					glog.Error(err)
				}
				c.chatWriter.Unlock()
			}
			return
		}
	}
	if c.eventWriter.Writer != nil {
		timestamp := append([]byte(time.Now().Format("R0102 15:04:05.000000")), []byte("] ")...)
		data = append([]byte(timestamp), data...)
		c.eventWriter.Lock()
		_, err := c.eventWriter.Write(data)
		if err != nil {
			raven.CaptureError(err, map[string]string{"app": "rcon", "module": "reader"})
			glog.Error(err)
		}
		c.eventWriter.Unlock()
	}
}
