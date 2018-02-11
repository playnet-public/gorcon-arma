package protocol

import (
	"testing"
)

func TestResponseType(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		data    []byte
		want    byte
		wantErr bool
	}{
		{
			"ok",
			[]byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			1,
			false,
		},
		{
			"error",
			[]byte{'B', 'E', 1, 1, 1, 1, 0},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResponseType(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResponseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResponseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSequence(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint32
		wantErr bool
	}{
		{
			"ok",
			[]byte{'B', 'E', 1, 1, 1, 1, 0, 1, 255, 10, 5, 2, 82},
			255,
			false,
		},
		{
			"error",
			[]byte{'B', 'E', 1, 1, 1, 1},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSequence(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSequence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetSequence() = %v, want %v", got, tt.want)
			}
		})
	}
}
