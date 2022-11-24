package types

import (
	"fmt"
	"io"
)

func ReadPrivBits(r io.Reader) PrivBits {
	var tmp uint16
	var rv PrivBits
	fmt.Fscanf(r, "%016b", &tmp)

	rv.Wheel = (tmp & 0x8000) == 0x8000
	rv.Admin = (tmp & 0x4000) == 0x4000
	rv.Statistic = (tmp & 0x2000) == 0x2000
	rv.CreatePersons = (tmp & 0x1000) == 0x1000
	rv.CreateConferences = (tmp & 0x0800) == 0x0800
	rv.ChangeName = (tmp & 0x0400) == 0x0400

	return rv
}

func ReadExtendedConfType(r io.Reader) ExtendedConfType {
	var tmp uint8
	var rv ExtendedConfType

	fmt.Fscanf(r, "%08b", &tmp)
	fmt.Printf("%02x\n", tmp)
	rv.RdProt = (tmp & 0x80) != 0
	rv.Original = (tmp & 0x40) != 0
	rv.Secret = (tmp & 0x20) != 0
	rv.LetterBox = (tmp & 0x10) != 0
	rv.AllowAnonymous = (tmp & 0x08) != 0
	rv.ForbidSecret = (tmp & 0x04) != 0
	rv.Reserved2 = (tmp & 0x02) != 0
	rv.Reserved3 = (tmp & 0x01) != 0

	return rv
}

func ReadPersonalFlags(r io.Reader) PersonalFlags {
	var tmp uint8
	var rv PersonalFlags

	fmt.Fscanf(r, "%08b", &tmp)

	rv.UnreadIsSecret = (tmp != 0)

	return rv
}

func ReadUInt32Array(r io.Reader) ([]uint32, error) {
	var rv []uint32

	var len int

	_, err := fmt.Fscanf(r, "%d", &len)
	if err != nil {
		return rv, err
	}

	fmt.Fscanf(r, " { ")
	for n := 0; n < len; n++ {
		var next uint32
		n, err := fmt.Fscanf(r, "%d ", &next)
		if n != 1 || err != nil {
			return rv, err
		}

		rv = append(rv, next)
	}

	_, err = fmt.Fscanf(r, "}")

	return rv, err
}

// func ReadTime(r io.Reader) time.Time {
// 	sec := int(readUInt32(r))
// 	min := int(readUInt32(r))
// 	hour := int(readUInt32(r))
// 	mday := int(readUInt32(r))
// 	mon := int(readUInt32(r))
// 	year := int(readUInt32(r))
// 	_ = int(readUInt32(r))
// 	_ = int(readUInt32(r))
// 	_ = int(readUInt32(r))
//
// 	return time.Date(1900+year, time.Month(mon), mday, hour, min, sec, 0, time.UTC)
// }
