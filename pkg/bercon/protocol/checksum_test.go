package protocol

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"testing"
)

func Test_getChecksum(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantC   uint32
		wantErr bool
		wantVal bool
	}{
		{
			"ok",
			[]byte{'B', 'E', 0, 1, 2, 3, 0xFF},
			(uint32(0) | uint32(1)<<8 | uint32(2)<<16 | uint32(3)<<24),
			false,
			true,
		},
		{
			"value_error",
			[]byte{'B', 'E', 2, 1, 2, 3, 0xFF},
			(uint32(0) | uint32(1)<<8 | uint32(2)<<16 | uint32(3)<<24),
			false,
			false,
		},
		{
			"data_error",
			[]byte{'B', 'E', 2, 1, 2, 0xFF},
			(uint32(0) | uint32(1)<<8 | uint32(2)<<16 | uint32(3)<<24),
			true,
			false,
		},
		{
			"header_error",
			[]byte{'C', 'E', 0, 1, 2, 3, 0xFF},
			(uint32(0) | uint32(1)<<8 | uint32(2)<<16 | uint32(3)<<24),
			true,
			false,
		},
		{
			"header_end_error",
			[]byte{'B', 'E', 0, 1, 2, 3, 0xF1},
			(uint32(0) | uint32(1)<<8 | uint32(2)<<16 | uint32(3)<<24),
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := getChecksum(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("getChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (gotC == tt.wantC) != tt.wantVal {
				t.Errorf("getChecksum() = %v, want %v", gotC, tt.wantC)
			}
		})
	}
}

func Test_CRCVerification(t *testing.T) {
	hash := []byte{37, 111, 118, 65}
	data := []byte{255, 0, 97, 100, 109, 105, 110}
	RealHash := binary.LittleEndian.Uint32(hash)
	ActualHash := crc32.Checksum(data, crc32.MakeTable(crc32.IEEE))
	liveExample := makeChecksum(data)
	livebinary := BuildLoginPacket("admin")[2:7]
	LivePacket := binary.LittleEndian.Uint32(livebinary)

	if RealHash != ActualHash {
		t.Error("Example Hash Does Not Match")
	}
	if RealHash != liveExample {
		t.Error("Hash Is not correctly Calculated in makeChecksum")
	}
	if RealHash != LivePacket {
		t.Error("Hash is not correctly Stored in Connection Packet\nExpected:", hash, "\nRecieved:", livebinary)
	}
	fmt.Println()
}

func Test_verifyChecksum(t *testing.T) {
	var TestData = []struct {
		Data     []byte
		CheckSum uint32
	}{
		{[]byte("kdokdwkdpoamdp10201"), crc32.ChecksumIEEE([]byte("kdokdwkdpoamdp10201"))},
		{[]byte("admin"), crc32.ChecksumIEEE([]byte("admin"))},
	}
	for _, v := range TestData {
		if verifyChecksum(v.Data, v.CheckSum) != true {
			t.Error("Test Data Failed")
		}

	}
}

func Test_verifyChecksumMatch(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantB   bool
		wantErr bool
	}{
		{
			"ok", //cmdpacket "tescmd"
			[]byte{66, 69, 49, 101, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			true,
			false,
		},
		{
			"ok", //cmdpacket "testcmd"
			[]byte{66, 69, 246, 169, 154, 223, 255, 1, 0, 116, 101, 115, 116, 99, 109, 100},
			true,
			false,
		},
		{
			"invalid_checksum", //cmdpacket "testcmd"
			[]byte{66, 69, 0, 169, 154, 223, 255, 1, 0, 116, 101, 115, 116, 99, 109, 100},
			false,
			true,
		},
		{
			"invalid_header", //cmdpacket "testcmd"
			[]byte{0, 69, 0, 169, 154, 223, 255, 1, 0, 116, 101, 115, 116, 99, 109, 100},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotB, err := verifyChecksumMatch(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyChecksumMatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotB != tt.wantB {
				t.Errorf("verifyChecksumMatch() = %v, want %v", gotB, tt.wantB)
			}
		})
	}
}
