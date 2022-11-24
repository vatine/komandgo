package utils

import (
	"testing"

	"github.com/vatine/komandgo/pkg/types"
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

func TestParseConfType(t *testing.T) {
	cases := []struct {
		source string
		offset int
		want   types.ConfType
	}{}

	for ix, c := range cases {
		got := ParseConfType(c.source, c.offset)

		if got.RdProt != c.want.RdProt {
			t.Errorf("Case #%d, RdProt saw %v, want %v", ix, got.RdProt, c.want.RdProt)
		}
		if got.Original != c.want.Original {
			t.Errorf("Case #%d, Original saw %v, want %v", ix, got.Original, c.want.Original)
		}
		if got.Secret != c.want.Secret {
			t.Errorf("Case #%d, Secret saw %v, want %v", ix, got.Secret, c.want.Secret)
		}
		if got.LetterBox != c.want.LetterBox {
			t.Errorf("Case #%d, LetterBox saw %v, want %v", ix, got.LetterBox, c.want.LetterBox)
		}
	}
}
