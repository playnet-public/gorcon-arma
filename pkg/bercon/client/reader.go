package client

import (
	"net"
	"strings"
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"go.uber.org/zap"
)

func (c *Client) readerLoop(ret chan error) {
	var err error
	c.RLock()
	defer c.RUnlock()
	defer func() { ret <- err }()
	for {
		c.log.Debug("looping reader")
		if !c.looping {
			c.log.Info("loop exited", zap.String("loop", "reader"))
			//TODO: Should we place some error return here?
			return
		}
		if c.con.UDPConn == nil {
			c.log.Error("invalid connection", zap.Error(common.ErrConnectionNil))
			err = common.ErrConnectionNil
			return
		}

		c.con.SetReadDeadline(time.Now().Add(time.Second * 2)) //Evaluate if Deadline is required
		n, err := c.con.Read(c.con.ReadBuffer)
		if err == nil {
			data := c.con.ReadBuffer[:n]
			c.log.Error("received data", zap.ByteString("data", data))
			if herr := c.handlePacket(data); herr != nil {
				c.log.Error("packet error", zap.Error(herr))
			}
			//TODO: Evaluate if parallel approach is better
			//go c.handlePacket(data)
		}
		if err != nil {
			if err, _ := err.(net.Error); err.Timeout() {
				c.log.Debug("timeout", zap.Error(err))
				continue
			} else {
				c.log.Debug("read error", zap.Error(err))
				return
			}
		}

	}
}

func (c *Client) handlePacket(packet []byte) error {
	seq, data, pType, err := protocol.VerifyPacket(packet)
	if err != nil {
		c.log.Debug("verify packet error", zap.Error(err))
		return err
	}

	// Handle Packet Types
	if pType == protocol.PacketType.ServerMessage {
		c.log.Debug("server message", zap.Uint32("seq", seq), zap.ByteString("data", data))
		c.handleServerMessage(append(data[3:], []byte("\n")...))
		if c.con.UDPConn != nil {
			c.con.SetWriteDeadline(time.Now().Add(time.Millisecond * 100))
			_, err := c.con.Write(protocol.BuildMsgAckPacket(seq))
			if err != nil {
				c.log.Debug("write ack error", zap.Error(err))
				return err
			}
		}
		return nil
	}

	if pType != protocol.PacketType.Command && pType != protocol.PacketType.MultiCommand {
		c.log.Debug("unknown packet", zap.Uint8("type", pType), zap.ByteString("packet", packet))
		return common.ErrUnknownPacketType
	}

	packetCount, currentPacket, isMultiPacket := protocol.CheckMultiPacketResponse(data)
	c.log.Debug("packet received", zap.Uint32("seq", seq), zap.Bool("multi", isMultiPacket), zap.ByteString("packet", data))
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
		if strings.HasPrefix(string(data), v) {
			if c.chatWriter.Writer != nil {
				c.chatWriter.Lock()
				_, err := c.chatWriter.Write(data)
				if err != nil {
					c.log.Error("chat write error", zap.Error(err))
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
			c.log.Error("event write error", zap.Error(err))
		}
		c.eventWriter.Unlock()
	}
}
