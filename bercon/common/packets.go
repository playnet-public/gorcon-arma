package common

func BuildPacket(data []byte, PacketType byte) []byte {
	data = append([]byte{0xFF, PacketType}, data...)
	checksum := makeChecksum(data)
	header := buildHeader(checksum)

	return append(header, data...)
}

func BuildLoginPacket(pw string) []byte {
	return BuildPacket([]byte(pw), PacketType.Login)
}

func BuildCmdPacket(cmd []byte, seq uint8) []byte {
	return BuildPacket(append([]byte{seq}, cmd...), PacketType.Command)
}

func BuildKeepAlivePacket(seq uint8) []byte {
	return BuildPacket([]byte{seq}, PacketType.Command)
}

func BuildMsgAckPacket(seq uint8) []byte {
	return BuildPacket([]byte{seq}, PacketType.ServerMessage)
}

func VerifyPacket(packet []byte) (seq byte, data []byte, pckType byte, err error) {
	checksum, err := getChecksum(packet)
	if err != nil {
		return
	}
	match := verifyChecksum(packet[6:], checksum)
	if !match {
		err = ErrInvalidChecksum
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

func VerifyLogin(packet []byte) (byte, error) {
	if len(packet) != 9 {
		return 0, ErrInvalidLoginPacket
	}
	if match, err := verifyChecksumMatch(packet); match == false || err != nil {
		return 0, ErrInvalidChecksum
	}

	return packet[8], nil
}

func CheckMultiPacketResponse(data []byte) (byte, byte, bool) {
	if len(data) < 3 {
		return 0, 0, false
	}
	if data[0] != 0x01 || data[2] != 0x00 {
		return 0, 0, false
	}
	return data[3], data[4], true
}
