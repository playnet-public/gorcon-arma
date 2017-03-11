package rcon

import "testing"

func Test_stripHeader(t *testing.T) {
	cmd := "Kick steve"
	pck := buildPacket([]byte(cmd), packetType.Command)
	result := stripHeader(pck)
	if string(result[2:]) != cmd {
		t.Fatal("Cmd Mismatch:", cmd, "-", string(result))
	}
}
