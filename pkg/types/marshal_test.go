package types

import (
	"testing"
)


func TestPrivBits(t *testing.T) {
	cases := []struct {
		privBits PrivBits
		expected string
	}{
		{PrivBits{Wheel: true}, "1000000000000000"},
		{PrivBits{}, "0000000000000000"},
		{PrivBits{Wheel: true, Statistic: true}, "1010000000000000"},
		{PrivBits{Wheel: true, Admin: true, ChangeName: true}, "1100010000000000"},
	}

	for ix, c := range cases {
		seen := c.privBits.Repr()
		if seen != c.expected {
			t.Errorf("case #%d, saw <%s> expected <%s>", ix, seen, c.expected)
		}
	}
}
