package bercon

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

func verifyPacket(packet []byte) (seq byte, data []byte, pckType byte, err error) {
	checksum, err := getChecksum(packet)
	if err != nil {
		return
	}
	match := verifyChecksum(packet[6:], checksum)
	if !match {
		err = ErrInvalidChecksum
		return
	}
	seq, err = getSequence(packet)
	if err != nil {
		return
	}
	data, err = stripHeader(packet)
	if err != nil {
		return
	}
	pckType, err = responseType(packet)
	return
}

func verifyLogin(packet []byte) (byte, error) {
	if len(packet) != 9 {
		return 0, ErrInvalidLoginPacket
	}
	if match, err := verifyChecksumMatch(packet); match == false || err != nil {
		return 0, ErrInvalidChecksum
	}

	return packet[8], nil
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
