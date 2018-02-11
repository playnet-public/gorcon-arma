package client

import (
	"reflect"
	"testing"

	"github.com/golang/glog"
)

func TestVerifyPacket(t *testing.T) {
	ch := makeChecksum([]byte{})
	glog.Errorf("%v", buildHeader(ch))
	tests := []struct {
		name        string
		packet      []byte
		wantSeq     uint32
		wantData    string
		wantPckType byte
		wantErr     bool
	}{
		{
			"ok",
			[]byte{66, 69, 49, 101, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			0,
			string([]byte{255, 1, 0, 116, 101, 115, 99, 109, 100}),
			PacketType.Command,
			false,
		},
		{
			"invalid_header",
			[]byte{66, 69, 0, 0, 0, 0},
			0,
			"",
			0,
			true,
		},
		{
			"invalid_packet",
			[]byte{0, 69, 49, 0, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			0,
			string([]byte{255, 1, 0, 116, 101, 115, 99, 109, 100}),
			0,
			true,
		},
		{
			"invalid_checksum",
			[]byte{66, 69, 49, 0, 26, 11, 255, 1, 0, 116, 101, 115, 99, 109, 100},
			0,
			string([]byte{255, 1, 0, 116, 101, 115, 99, 109, 100}),
			0,
			true,
		},
		{
			"invalid_seq",
			[]byte{66, 69, 27, 223, 250, 165, 255, 1},
			0,
			string([]byte{255, 1}),
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeq, gotData, gotPckType, err := VerifyPacket(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSeq != tt.wantSeq {
				t.Errorf("VerifyPacket() gotSeq = %v, want %v", gotSeq, tt.wantSeq)
			}
			if string(gotData) != tt.wantData {
				t.Errorf("VerifyPacket() gotData = %v, want %v", gotData, tt.wantData)
			}
			if gotPckType != tt.wantPckType {
				t.Errorf("VerifyPacket() gotPckType = %v, want %v", gotPckType, tt.wantPckType)
			}
		})
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

func TestCheckMultiPacketResponse(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  byte
		want1 byte
		want2 bool
	}{
		{
			"basic",
			[]byte{1, 0, 116, 101, 115, 99, 109, 100},
			0,
			0,
			false,
		},
		{
			"invalid_data",
			[]byte{1, 0},
			0,
			0,
			false,
		},
		{
			"multi_packet",
			[]byte{1, 1, 0, 4, 3, 99, 109, 100},
			4,
			3,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := CheckMultiPacketResponse(tt.data)
			if got != tt.want {
				t.Errorf("CheckMultiPacketResponse() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("CheckMultiPacketResponse() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("CheckMultiPacketResponse() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestBuildPackets(t *testing.T) {
	tests := []struct {
		name string
		seq  uint32
		data []byte
		t    byte
		want []byte
	}{
		{
			"login",
			0,
			[]byte("pw"),
			PacketType.Login,
			[]byte{66, 69, 132, 68, 31, 30, 255, 0, 112, 119},
		},
		{
			"keep_alive",
			0,
			[]byte{},
			PacketType.Command,
			[]byte{66, 69, 190, 220, 194, 88, 255, 1, 0},
		},
		{
			"cmd",
			0,
			[]byte("xxx"),
			PacketType.Command,
			[]byte{66, 69, 199, 16, 188, 139, 255, 1, 0, 120, 120, 120},
		},
		{
			"ack",
			0,
			[]byte{},
			PacketType.ServerMessage,
			[]byte{66, 69, 125, 143, 239, 115, 255, 2, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.t {
			case PacketType.Login:
				if got := BuildLoginPacket(string(tt.data)); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("BuildLoginPacket() = %v, want %v", got, tt.want)
				}
			case PacketType.Command:
				if len(tt.data) < 1 {
					if got := BuildKeepAlivePacket(tt.seq); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("BuildKeepAlivePacket() = %v, want %v", got, tt.want)
					}
				} else {
					if got := BuildCmdPacket(tt.data, tt.seq); !reflect.DeepEqual(got, tt.want) {
						t.Errorf("BuildCmdPacket() = %v, want %v", got, tt.want)
					}
				}
			case PacketType.ServerMessage:
				if got := BuildMsgAckPacket(tt.seq); !reflect.DeepEqual(got, tt.want) {
					t.Errorf("BuildMsgAckPacket() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBuildKeepAlivePacket(t *testing.T) {
	type args struct {
		seq uint32
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildKeepAlivePacket(tt.args.seq); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildKeepAlivePacket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildCmdPacket(t *testing.T) {
	type args struct {
		cmd []byte
		seq uint32
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildCmdPacket(tt.args.cmd, tt.args.seq); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildCmdPacket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildLoginPacket(t *testing.T) {
	type args struct {
		pw string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildLoginPacket(tt.args.pw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildLoginPacket() = %v, want %v", got, tt.want)
			}
		})
	}
}
