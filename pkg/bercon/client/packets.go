package client

import (
	"fmt"

	raven "github.com/getsentry/raven-go"
	"github.com/playnet-public/gorcon-arma/pkg/common"
)

//BuildPacket creates a new packet with data and type
func BuildPacket(data []byte, PacketType byte) []byte {
	data = append([]byte{0xFF, PacketType}, data...)
	checksum := makeChecksum(data)
	header := buildHeader(checksum)

	return append(header, data...)
}

//BuildLoginPacket creates a login packet with password
func BuildLoginPacket(pw string) []byte {
	return BuildPacket([]byte(pw), PacketType.Login)
}

//BuildCmdPacket creates a packet with cmd and seq
func BuildCmdPacket(cmd []byte, seq uint8) []byte {
	return BuildPacket(append([]byte{seq}, cmd...), PacketType.Command)
}

//BuildKeepAlivePacket creates a keepAlivePacket with seq
func BuildKeepAlivePacket(seq uint8) []byte {
	return BuildPacket([]byte{seq}, PacketType.Command)
}

//BuildMsgAckPacket creates a server message packet with seq
func BuildMsgAckPacket(seq uint8) []byte {
	return BuildPacket([]byte{seq}, PacketType.ServerMessage)
}

//VerifyPacket checks a package and its contents for errors or tampering
func VerifyPacket(packet []byte) (seq byte, data []byte, pckType byte, err error) {
	defer func() {
		if err != nil {
			raven.CaptureError(fmt.Errorf("%v - Packet: %v", err, string(data)), map[string]string{"app": "rcon", "module": "packets"})
		}
	}()
	checksum, err := getChecksum(packet)
	if err != nil {
		return
	}
	match := verifyChecksum(packet[6:], checksum)
	if !match {
		err = common.ErrInvalidChecksum
		return
	}
	seq, err = GetSequence(packet)
	if err != nil {
		return
	}
	data, err = stripHeader(packet)
	if err != nil {
		return
	}
	pckType, err = ResponseType(packet)
	return
}

//VerifyLogin checks the login packet
func VerifyLogin(packet []byte) (b byte, err error) {
	b = 0
	defer func() {
		if err != nil {
			raven.CaptureError(err, nil)
		}
	}()
	if len(packet) != 9 {
		err = common.ErrInvalidLoginPacket
		return b, err
	}
	var match bool
	if match, err = verifyChecksumMatch(packet); match == false || err != nil {
		return b, err
	}

	return packet[8], nil
}

//CheckMultiPacketResponse checks whether a packet is part of a multiPacketResponse
func CheckMultiPacketResponse(data []byte) (byte, byte, bool) {
	if len(data) < 3 {
		return 0, 0, false
	}
	if data[0] != 0x01 || data[2] != 0x00 {
		return 0, 0, false
	}
	return data[3], data[4], true
}
