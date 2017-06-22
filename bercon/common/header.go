package common

import (
	"encoding/binary"

	raven "github.com/getsentry/raven-go"
)

func buildHeader(checksum uint32) []byte {
	check := make([]byte, 4)
	binary.LittleEndian.PutUint32(check, checksum)
	return append([]byte{}, 'B', 'E', check[0], check[1], check[2], check[3])
}

func stripHeader(data []byte) ([]byte, error) {
	if len(data) < 7 {
		raven.CaptureError(ErrInvalidSizeNoHeader, nil)
		return []byte{}, ErrInvalidSizeNoHeader
	}
	return data[6:], nil
}
