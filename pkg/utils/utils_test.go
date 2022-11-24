package utils

import (
	"testing"
)

func TestReadUInt32FromString(t *testing.T) {
	cases := []struct {
		in         string
		offset     int
		wantNum    uint32
		wantOffset int
	}{
		{"123", 0, 123, 3},
		{"123", 1, 23, 3},
		{"123", 2, 3, 3},
		{"123 321", 0, 123, 3},
		{"123 321", 3, 321, 7},
		{"123 321", 4, 321, 7},
	}

	for ix, c := range cases {
		got, gotOffset := ReadUInt32FromString(c.in, c.offset)
		if got != c.wantNum {
			t.Errorf("Case #%d, got %d, want %d", ix, got, c.wantNum)
		}
		if gotOffset != c.wantOffset {
			t.Errorf("Case #%d, got offset %d, want %d", ix, gotOffset, c.wantOffset)
		}
	}
}

func TestParseConfType(in string, offset int) types.ConfType {
	cases := []struct {
		source string
		offset int
		want   types.ConfType
	}{}

	for ix, c := range cases {
		got := ParseConfType(c.source, c.offset)

		if got.RdOnly != c.want.RdOnly {
			t.Errorf("Case #%d, RdOnly saw %v, want %v", ix, got.RdOnly, c.want.RdOnly)
		}
		if got.Original != c.want.Original {
			t.Errorf("Case #%d, Original saw %v, want %v", ix, got.Original, c.want.Original)
		}
		if got.Secret != c.want.Secret {
			t.Errorf("Case #%d, Secret saw %v, want %v", ix, got.Secret, c.want.Secret)
		}
		if got.Mailbox != c.want.Mailbox {
			t.Errorf("Case #%d, Mailbox saw %v, want %v", ix, got.Mailbox, c.want.Mailbox)
		}
	}
}
