package protocol

import (
	"hash/crc32"

	raven "github.com/getsentry/raven-go"
	"github.com/playnet-public/gorcon-arma/pkg/common"
)

func makeChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func getChecksum(data []byte) (c uint32, err error) {
	c = 0
	if len(data) < 7 {
		err = common.ErrInvalidHeaderSize
		return
	}
	if data[0] != 'B' || data[1] != 'E' {
		err = common.ErrInvalidHeaderSyntax
		return
	}
	if data[6] != 0xFF {
		err = common.ErrInvalidHeaderEnd
		return
	}
	c = uint32(data[2]) | uint32(data[3])<<8 | uint32(data[4])<<16 | uint32(data[5])<<24
	return
}

func verifyChecksum(data []byte, checksum uint32) bool {
	sum := crc32.ChecksumIEEE(data)
	if sum != checksum {
		return false
	}
	return true
}

func verifyChecksumMatch(data []byte) (b bool, err error) {
	b = false
	defer func() {
		if err != nil {
			raven.CaptureError(err, nil)
		}
	}()
	checksum, err := getChecksum(data)
	if err != nil {
		return false, err
	}
	match := verifyChecksum(data[6:], checksum)
	if !match {
		return false, common.ErrInvalidChecksum
	}
	return true, nil
}
