package bercon

import (
	"encoding/binary"
	"testing"
)

func Test_buildHeader(t *testing.T) {
	TestValues := []uint32{
		58, 25, 1400, 980, 4294967295, 0, 2147483647, 1600581284, 3848910246, 108500257,
	}
	a := make([]byte, 4)
	for _, v := range TestValues {
		binary.LittleEndian.PutUint32(a, v)
		h := buildHeader(v)
		if len(h) != 6 {
			t.Error("Header Invalid Size")
		}
		if h[0] != 'B' || h[1] != 'E' || h[2] != a[0] || h[3] != a[1] || h[4] != a[2] || h[5] != a[3] {
			t.Error("Header Signature Not Correct")
		}
	}
}

func Test_stripHeader(t *testing.T) {
	cmd := "Kick steve"
	packet := buildPacket([]byte(cmd), packetType.Command)
	result, err := stripHeader(packet)
	if err != nil {
		t.Fatal("on StripHeader:", err.Error())
	}
	if string(result[2:]) != cmd {
		t.Fatal("Expected:", cmd, "Got:", string(result))
	}
}
