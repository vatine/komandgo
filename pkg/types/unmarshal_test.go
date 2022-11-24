package types

import (
	"testing"

	"strings"
)

func TestReadPrivBits(t *testing.T) {
	cases := []struct {
		in string
		pb PrivBits
	}{
		{"1000000000000000", PrivBits{Wheel: true}},
		{"0000000000000000", PrivBits{}},
		{"1010000000000000", PrivBits{Wheel: true, Statistic: true}},
		{"1100010000000000", PrivBits{Wheel: true, Admin: true, ChangeName: true}},
	}

	for ix, c := range cases {
		r := strings.NewReader(c.in)
		saw := ReadPrivBits(r)
		want := c.pb
		if saw != want {
			t.Errorf("Case #%d, saw %v, want %v", ix, saw, want)
		}
	}
}

func TestReadExtendedConfType(t *testing.T) {
	cases := []struct {
		in string
		ec ExtendedConfType
	}{
		{
			"01011000",
			ExtendedConfType{
				Original:       true,
				LetterBox:      true,
				AllowAnonymous: true,
			},
		},
		{"00000001", ExtendedConfType{Reserved3: true}},
	}

	for ix, c := range cases {
		r := strings.NewReader(c.in)
		want := c.ec
		saw := ReadExtendedConfType(r)
		if saw != want {
			t.Errorf("Case #%d: saw %v, want %v", ix, saw, want)
		}
	}
}
