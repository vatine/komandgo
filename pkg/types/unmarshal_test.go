package types

import (
	"testing"

	"strings"
)

func cmpSlice(got []uint32, want []uint32, t *testing.T) bool {
	if len(got) != len(want) {
		t.Errorf("slice lengths differ")
		return false
	}

	for ix, gVal := range got {
		wVal := want[ix]

		if gVal != wVal {
			t.Errorf("at index %d, got %d and want %d", ix, gVal, wVal)
			return false
		}
	}

	return true
}

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

func TestReadUint32Array(t *testing.T) {
	cases := []struct {
		in   string
		want []uint32
		err  bool
	}{
		{"apa", []uint32{}, true},
		{"1 { 4711 }", []uint32{4711}, false},
		{"1 { zig }", []uint32{}, true},
		{"2 { 4711 65 }", []uint32{4711, 65}, false},
		{"3 { 4711 65 }", []uint32{4711, 65}, true},
		{"1 { 4711 65 }", []uint32{4711, 65}, true},
	}

	for ix, tc := range cases {
		r := strings.NewReader(tc.in)
		got, err := ReadUInt32Array(r)

		if err != nil {
			if !tc.err {
				t.Errorf("Case %d, unexpected error %v", ix, err)
			}
		} else {
			if tc.err {
				t.Errorf("Case %d, expected an error, saw none", ix)
			}
			if !cmpSlice(got, tc.want, t) {
				t.Errorf("Case %d, got %v, want %v", ix, got, tc.want)
			}
		}
	}
}
