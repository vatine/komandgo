// Create string representation of various types, usable for on-the-wire
package types

import (
	"fmt"
	"strings"
	"time"
)

func (ec ExtendedConfType) Repr() string {
	tmp := uint8(0)

	if ec.RdProt {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.Original {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.Secret {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.LetterBox {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.AllowAnonymous {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.ForbidSecret {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.Reserved2 {
		tmp |= 1
	}
	tmp = tmp << 1
	if ec.Reserved3 {
		tmp |= 1
	}

	return fmt.Sprintf("%08b", tmp)
}

func (p PrivBits) Repr() string {
	tmp := uint16(0)

	if p.Wheel {
		tmp |= 1
	}

	tmp <<= 1
	if p.Admin {
		tmp |= 1
	}

	tmp <<= 1
	if p.Statistic {
		tmp |= 1
	}

	tmp <<= 1
	if p.CreatePersons {
		tmp |= 1
	}

	tmp <<= 1
	if p.CreateConferences {
		tmp |= 1
	}

	tmp <<= 1
	if p.ChangeName {
		tmp |= 1
	}

	tmp <<= 10

	return fmt.Sprintf("%016b", tmp)
}

func TextNoArray(ts []TextNo) string {
	var b strings.Builder
	w := &b

	fmt.Fprintf(w, "%d { ", len(ts))
	for _, v := range ts {
		fmt.Fprintf(w, "%d ", v)
	}
	fmt.Fprintf(w, "}")

	return w.String()
}

func UInt32Array(ar []uint32) string {
	var b strings.Builder
	w := &b

	fmt.Fprintf(w, "%d { ", len(ar))
	for _, v := range ar {
		fmt.Fprintf(w, "%d ", v)
	}
	fmt.Fprintf(w, "}")

	return w.String()
}

func StringTime(when time.Time) string {
	sec := when.Second()
	min := when.Minute()
	hour := when.Hour()
	mday := when.Day()
	mon := when.Month() - 1
	wday := when.Weekday()
	yday := when.YearDay() - 1
	year := when.Year() - 1900
	isdst := 0
	if when.IsDST() {
		isdst = 1
	}

	return fmt.Sprintf("%d %d %d %d %d %d %d %d %d", sec, min, hour, mday, mon, year, wday, yday, isdst)
}
