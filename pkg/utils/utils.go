package utils

import (
	"io"
)

// Read a single byte from an io.Reader
func ReadByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return 0, err
		}
		if n == 1 {
			return buf[0], nil
		}
	}

	return 0, nil
}
