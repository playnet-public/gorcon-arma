package protocol

import (
	"encoding/binary"
	"reflect"
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

func Test_sstripHeader(t *testing.T) {
	cmd := "Kick steve"
	packet := BuildPacket([]byte(cmd), PacketType.Command)
	result, err := stripHeader(packet)
	if err != nil {
		t.Fatal("on StripHeader:", err.Error())
	}
	if string(result[2:]) != cmd {
		t.Fatal("Expected:", cmd, "Got:", string(result))
	}
}

func Test_stripHeader(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    []byte
		wantErr bool
	}{
		{
			"ok",
			[]byte{66, 69, 49, 101, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			[]byte{255, 1, 0, 116, 101, 115, 99, 109, 100},
			false,
		},
		{
			"invalid_data",
			[]byte{66, 69, 49, 101, 26, 11},
			[]byte{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripHeader(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("stripHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}
