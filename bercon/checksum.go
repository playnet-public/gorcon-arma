package bercon

import (
	"hash/crc32"

	"github.com/golang/glog"
)

func makeChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func getChecksum(data []byte) (uint32, error) {
	if len(data) < 7 {
		return 0, ErrInvalidHeaderSize
	}
	if data[0] != 'B' || data[1] != 'E' {
		return 0, ErrInvalidHeaderSyntax
	}
	if data[6] != 0xFF {
		return 0, ErrInvalidHeaderEnd
	}
	checksum := uint32(data[2]) | uint32(data[3])<<8 | uint32(data[4])<<16 | uint32(data[5])<<24
	return checksum, nil
}

func verifyChecksum(data []byte, checksum uint32) bool {
	return crc32.ChecksumIEEE(data) == checksum
}

func verifyChecksumMatch(data []byte) (bool, error) {
	checksum, err := getChecksum(data)
	if err != nil {
		glog.V(3).Infoln("verifyChecksumMatch: failed to get checksum")
		return false, err
	}
	match := verifyChecksum(data[6:], checksum)
	if !match {
		glog.V(3).Infoln("verifyChecksumMatch: failed at checksum match")
		return false, nil
	}
	return true, nil
}
