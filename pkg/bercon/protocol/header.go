package protocol

import (
	"encoding/binary"

	"github.com/playnet-public/gorcon-arma/pkg/common"
)

func buildHeader(checksum uint32) []byte {
	check := make([]byte, 4)
	binary.LittleEndian.PutUint32(check, checksum)
	return append([]byte{}, 'B', 'E', check[0], check[1], check[2], check[3])
}

func stripHeader(data []byte) ([]byte, error) {
	if len(data) < 7 {
		return []byte{}, common.ErrInvalidSizeNoHeader
	}
	return data[6:], nil
}
