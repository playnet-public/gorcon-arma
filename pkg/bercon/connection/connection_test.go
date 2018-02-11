package connection

import (
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/bercon/protocol"
)

func TestNew(t *testing.T) {
	c := New()
	tests := []struct {
		name string
		want *Conn
	}{
		{
			"basic",
			c,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConn_Connect(t *testing.T) {
	srv, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Errorf("failed to open udp server: %v", srv.LocalAddr())
	}
	addr, err := net.ResolveUDPAddr("udp", srv.LocalAddr().String())
	if err != nil {
		t.Errorf("failed to resolve address: %v", srv.LocalAddr())
	}
	con := New()
	type args struct {
		addr *net.UDPAddr
	}
	tests := []struct {
		name    string
		c       *Conn
		args    args
		wantErr bool
	}{
		{
			"ok",
			con,
			args{
				addr,
			},
			false,
		},
		{
			"error",
			con,
			args{
				nil,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.Connect(tt.args.addr); (err != nil) != tt.wantErr {
				t.Errorf("Conn.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConn_Login(t *testing.T) {
	srv, err := net.ListenPacket("udp", "")
	if err != nil {
		t.Errorf("failed to open udp server: %v", srv.LocalAddr())
	}
	addr, err := net.ResolveUDPAddr("udp", srv.LocalAddr().String())
	if err != nil {
		t.Errorf("failed to resolve address: %v", srv.LocalAddr())
	}
	con := New()
	con.Connect(addr)

	tests := []struct {
		name    string
		c       *Conn
		pass    string
		wantErr bool
	}{
		{
			"ok",
			con,
			"test",
			false,
		},
		{
			"error",
			con,
			"error",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exit := make(chan bool)
			go func() {
				err := tt.c.Login(tt.pass)
				if (err != nil) != tt.wantErr {
					t.Errorf("Conn.Login() error = %v, wantErr %v", err, tt.wantErr)
				}
				exit <- true
			}()
		Loop:
			for {
				select {
				case <-exit:
					break
				default:
					buffer := make([]byte, 4069)
					srv.SetReadDeadline(time.Now().Add(time.Second * 1))
					i, _, _ := srv.ReadFrom(buffer)
					fmt.Println(buffer[:i])
					if string(buffer[:i]) == string(protocol.BuildLoginPacket("error")) {
						fmt.Println("skipping error login")
						<-exit
						break Loop
					}
					if string(buffer[:i]) == string(protocol.BuildLoginPacket("test")) {
						fmt.Println("sending test packet")
						_, err := srv.WriteTo([]byte{66, 69, 40, 236, 197, 47, 255, 1, 0x01}, con.LocalAddr())
						if err != nil {
							t.Errorf("udp write error %v", err)
						}
						<-exit
						break Loop
					}
					if string(buffer[:i]) == string(protocol.BuildLoginPacket("invalid")) {
						fmt.Println("sending invalidLogin packet")
						_, err := srv.WriteTo([]byte{66, 69, 190, 220, 194, 88, 255, 1, 0x00}, con.LocalAddr())
						if err != nil {
							t.Errorf("udp write error %v", err)
						}
						<-exit
						break Loop
					}
				}
			}
			fmt.Println("running next test:", tt.name)
		})
	}
}

func TestConn_Transmission(t *testing.T) {
	con := New()
	con.SetTransmission(0, protocol.Transmission{
		Packet: []byte("test"),
	})
	con.SetTransmission(1, protocol.Transmission{
		Packet: []byte("test1"),
	})
	con.DeleteTransmission(1)
	tests := []struct {
		name  string
		seq   uint32
		want  protocol.Transmission
		want1 bool
	}{
		{
			"ok",
			0,
			protocol.Transmission{
				Packet: []byte("test"),
			},
			true,
		},
		{
			"deleted",
			1,
			protocol.Transmission{},
			false,
		},
		{
			"notOK",
			6,
			protocol.Transmission{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := con.GetTransmission(tt.seq)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Conn.GetTransmission() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Conn.GetTransmission() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestConn_Sequence(t *testing.T) {
	con := New()

	if x := con.Sequence(); x != 0 {
		t.Error("seq has to be 0, was", x)
	}

	con.AddSequence()
	if x := con.Sequence(); x != 1 {
		t.Error("seq has to be 1, was", x)
	}
}
func TestConn_Pingback(t *testing.T) {
	con := New()

	if x := con.Pingback(); x != 0 {
		t.Error("pingback has to be 0, was", x)
	}

	con.AddPingback()
	if x := con.Pingback(); x != 1 {
		t.Error("pingback has to be 1, was", x)
	}
	con.ResetPingback()
	if x := con.Pingback(); x != 0 {
		t.Error("pingback has to be 0, was", x)
	}
}
func TestConn_KeepAlive(t *testing.T) {
	con := New()

	if x := con.KeepAlive(); x != 0 {
		t.Error("keepAlive has to be 0, was", x)
	}

	con.AddKeepAlive()
	if x := con.KeepAlive(); x != 1 {
		t.Error("keepAlive has to be 1, was", x)
	}
	con.ResetKeepAlive()
	if x := con.KeepAlive(); x != 0 {
		t.Error("keepAlive has to be 1, was", x)
	}
}
