package utils

import (
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/vatine/komandgo/pkg/types"
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

func ReadUInt32FromString(s string, start int) (uint32, int) {
	var rv uint32
	done := false
	started := false
	for !done {
		if start >= len(s) {
			done = true
			continue
		}

		c := s[start]
		p := strings.IndexByte("0123456789", c)
		start++
		if p >= 0 {
			started = true
			rv = 10*rv + uint32(p)
		} else {
			if started {
				done = true
				start--
			}
		}
	}

	return rv, start
}

// Read from a stream until we've read a full delimited list, then
// return a string that is composed of the read bytes. If the start is
// sent in as 0, we skip matching the start. On error, simply return
// what's been read so far and the error seen
func ReadDelimitedList(start, end byte, r io.Reader) (string, error) {
	var rv []byte

	b, err := ReadByte(r)
	if err != nil || (start != 0 && b != start) {
		log.WithFields(log.Fields{
			"b":     b,
			"start": start,
			"end":   end,
		}).Error("Unexpected start of list")
		return "", fmt.Errorf("Unexpected start '%c'", b)
	}
	rv = append(rv, b)

	for {
		b, err := ReadByte(r)
		if err != nil {
			return string(rv), err
		}
		rv = append(rv, b)
		if b == end {
			return string(rv), nil
		}
	}

	return "", nil
}

func ParseConfType(s string, start int) types.ConfType {
	var rv types.ConfType

	rv.RdProt = (s[start+0] == '1')
	rv.Original = (s[start+1] == '1')
	rv.Secret = (s[start+2] == '1')
	rv.LetterBox = (s[start+3] == '1')

	return rv
}
