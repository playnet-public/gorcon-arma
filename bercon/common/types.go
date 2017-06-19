package common

//PacketType contains all possible types a packet could have
var PacketType = struct {
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

//PacketResponse contains all types a server could respond with
var PacketResponse = struct {
	LoginOk      byte
	LoginFail    byte
	MultiCommand byte
}{
	LoginOk:      0x01,
	LoginFail:    0x00,
	MultiCommand: 0x00,
}

//ResponseType gets the PacketResponse from a packet
func ResponseType(data []byte) (byte, error) {
	if len(data) < 8 {
		return 0, ErrInvalidSize
	}
	return data[7], nil
}

//GetSequence extracts the seq number from a packet
func GetSequence(data []byte) (byte, error) {
	if len(data) < 9 {
		return 0, ErrInvalidSizeNoSequence
	}
	return data[8], nil
}
