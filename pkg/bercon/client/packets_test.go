package client

import (
	"testing"
)

func Test_getSequence(t *testing.T) {
	var tests = []struct {
		test     []byte
		expected byte
	}{
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			expected: 255,
		},
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 85},
			expected: 85,
		},
	}

	for _, v := range tests {
		result, err := GetSequence(v.test)
		if err != nil {
			t.Error("Packet Size mismatch")
		}
		if result != v.expected {
			t.Error("Expected:", v.expected, "Got:", result)
		}
	}
}

func Test_responseType(t *testing.T) {
	var tests = []struct {
		test     []byte
		expected byte
	}{
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			expected: 1,
		},
		{
			test:     []byte{'B', 'E', 1, 1, 1, 1, 0, 1, 85},
			expected: 1,
		},
	}
	for _, v := range tests {
		result, err := ResponseType(v.test)
		if err != nil {
			t.Error("Test:", v.test, "Failed due to error:", err)
		}
		if result != v.expected {
			t.Error("Expected:", v.expected, "Got:", result)
		}
	}
}

func TestVerifyLogin(t *testing.T) {
	tests := []struct {
		name    string
		packet  []byte
		wantB   byte
		wantErr bool
	}{
		{
			"ok",
			[]byte{66, 69, 40, 236, 197, 47, 255, 1, 0x01},
			PacketResponse.LoginOk,
			false,
		},
		{
			"fail",
			[]byte{66, 69, 190, 220, 194, 88, 255, 1, 0x00},
			PacketResponse.LoginFail,
			false,
		},
		{
			"error",
			[]byte{0, 69, 49, 0, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			0,
			true,
		},
		{
			"checksum_error",
			[]byte{66, 69, 0, 220, 194, 88, 255, 1, 0x00},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotB, err := VerifyLogin(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotB != tt.wantB {
				t.Errorf("VerifyLogin() = %v, want %v", gotB, tt.wantB)
			}
		})
	}
}
