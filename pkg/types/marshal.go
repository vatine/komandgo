// Create string representation of various types, usable for on-the-wire
package types

import (
	"fmt"
)

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
