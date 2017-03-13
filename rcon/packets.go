package rcon

import (
	"encoding/binary"
	"errors"
	"hash/crc32"

	"github.com/golang/glog"
)

var packetType = struct {
	Login         byte
	Command       byte
	MultiCommand  byte
	ServerMessage byte
}{
	Login:         0x00,
	Command:       0x01,
	MultiCommand:  0x00,
	ServerMessage: 0x02,
}

var packetResponse = struct {
	LoginOk      byte
	LoginFail    byte
	MultiCommand byte
}{
	LoginOk:      0x01,
	LoginFail:    0x00,
	MultiCommand: 0x00,
}

func buildHeader(checksum uint32) []byte {
	check := make([]byte, 4)
	binary.LittleEndian.PutUint32(check, checksum)
	return append([]byte{}, 'B', 'E', check[0], check[1], check[2], check[3])
}

func stripHeader(data []byte) ([]byte, error) {
	if len(data) < 7 {
		return []byte{}, errors.New("Invalid Packet Size, no Header found")
	}
	return data[6:], nil
}

func makeChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func getChecksum(data []byte) (uint32, error) {
	if len(data) < 7 {
		return 0, errors.New("Invalid Packet Header Size")
	}
	if data[0] != 'B' || data[1] != 'E' {
		return 0, errors.New("Invalid Packet Header Syntax")
	}
	if data[6] != 0xFF {
		return 0, errors.New("Invalid Packet Header end")
	}
	checksum := uint32(data[2]) | uint32(data[3])<<8 | uint32(data[4])<<16 | uint32(data[5])<<24
	return checksum, nil
}

func verifyChecksum(data []byte, checksum uint32) bool {
	return crc32.ChecksumIEEE(data) == checksum
}

func verifyChecksumMatch(data []byte) (bool, error) {
	checksum, err := getChecksum(data)
	if err != nil {
		glog.V(3).Infoln("verifyChecksumMatch: failed to get checksum") // TODO: Verify if required
		return false, err
	}
	match := verifyChecksum(data[6:], checksum)
	if !match {
		glog.V(3).Infoln("verifyChecksumMatch: failed at checksum match")
		return false, nil
	}
	return true, nil
}

func getSequence(data []byte) byte {
	//TODO: Evaluate len check
	return data[8]
}

func buildPacket(data []byte, PacketType byte) []byte {
	data = append([]byte{0xFF, PacketType}, data...)
	checksum := makeChecksum(data)
	header := buildHeader(checksum)

	return append(header, data...)
}

func buildLoginPacket(pw string) []byte {
	return buildPacket([]byte(pw), packetType.Login)
}

func buildCmdPacket(cmd []byte, seq uint8) []byte {
	return buildPacket(append([]byte{seq}, cmd...), packetType.Command)
}

func buildKeepAlivePacket(seq uint8) []byte {
	return buildPacket([]byte{seq}, packetType.Command)
}

func buildMsgAckPacket(seq uint8) []byte {
	return buildPacket([]byte{seq}, packetType.ServerMessage)
}

func responseType(data []byte) (byte, error) {
	if len(data) < 8 {
		return 0, errors.New("Packet size too small")
	}
	return data[7], nil
}

func verifyPacket(packet []byte) (seq byte, data []byte, pckType byte, err error) {
	checksum, err := getChecksum(packet)
	if err != nil {
		glog.V(3).Infoln("verifyPacket: failed to get checksum") // TODO: Verify if required
		return
	}
	match := verifyChecksum(packet[6:], checksum)
	if !match {
		glog.V(3).Infoln("verfiyPacket: failed at checksum match")
		err = errors.New("Checksum does not match data")
		return
	}
	seq = getSequence(packet)
	data, err = stripHeader(packet)
	if err != nil {
		return
	}
	pckType, err = responseType(packet)
	return
}

func checkMultiPacketResponse(data []byte) (byte, byte, bool) {
	if len(data) < 3 {
		return 0, 0, false
	}
	if data[0] != 0x01 || data[2] != 0x00 {
		return 0, 0, false
	}
	return data[3], data[4], true
}
