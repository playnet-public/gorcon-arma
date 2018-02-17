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

		data, err := c.con.Read()
		if err, ok := err.(net.Error); ok && err.Timeout() {
			c.log.Debug("timeout", zap.Error(err))
			continue
		}
		if err != nil {
			c.log.Error("read error", zap.Error(err))
			return
		}

		c.log.Debug("received data", zap.ByteString("data", data))
		if err := c.handlePacket(data); err != nil {
			c.log.Error("packet error", zap.Error(err))
		}
		//TODO: Evaluate if parallel approach is better
		//go c.handlePacket(data)

	}
}

func (c *Client) handlePacket(packet []byte) (err error) {
	seq, data, pType, err := protocol.VerifyPacket(packet)
	if err != nil {
		c.log.Debug("verify packet error", zap.ByteString("packet", packet), zap.Error(err))
		return err
	}

	switch pType {
	case protocol.PacketType.ServerMessage:
		return c.handleServerMessage(seq, data)

	case protocol.PacketType.Command:
		c.handleResponse(seq, data[3:], true)
		return nil

	case protocol.PacketType.MultiCommand:
		return c.handleMultiPacket(seq, data)

	default:
		c.log.Debug("unknown packet", zap.Uint8("type", pType), zap.ByteString("packet", packet))
		return common.ErrUnknownPacketType
	}
	return common.ErrUnknownPacketType
}

func (c *Client) handleMultiPacket(seq uint32, data []byte) error {
	packetCount, currentPacket, isSingle := protocol.CheckMultiPacketResponse(data)
	c.log.Debug("packet received", zap.Uint32("seq", seq), zap.Bool("single", isSingle), zap.ByteString("packet", data))
	last := (currentPacket+1 >= packetCount)
	c.handleResponse(seq, data[6:], last)
	return nil
}

func (c *Client) handleServerMessage(seq uint32, data []byte) error {
	c.log.Debug("server message", zap.Uint32("seq", seq), zap.ByteString("data", data))
	data = append(data[3:], []byte("\n")...)
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
			return c.con.WriteAck(seq)
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
	return c.con.WriteAck(seq)
}
