package protocol

// Various client implementation tests

import (
	"testing"

	"bytes"
	"strings"

	"github.com/sirupsen/logrus"
)

func TestReadUInt(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	td := []struct {
		data     string
		expected uint32
	}{
		{"2 ", 2}, {"2 22", 2}, {"22 ", 22}, {"012", 12},
		{"0 ", 0}, {"990099 ", 990099},
	}

	for ix, d := range td {
		seen := readUInt32(strings.NewReader(d.data))
		expected := d.expected
		if seen != expected {
			t.Errorf("Case #%d, saw %d, expected %d", ix, seen, expected)
		}
	}
}

// Create a fake client, with specific data to read
func fakeClient(data string) *KomClient {
	return &KomClient{
		socket:   bytes.NewBufferString(data),
		asyncMap: make(map[uint32]Callback),
		shutdown: make(chan struct{}),
	}
}

func TestReadOKAndError(t *testing.T) {
	td := []struct {
		data string
		err  bool
	}{
		{"=1\n", false}, {"%1 2 3\n", true},
	}

	for ix, test := range td {
		c := fakeClient(test.data)
		rv := make(chan genericResponse)
		c.asyncMap[1] = genericCallback(rv)
		go c.receiveLoop()
		seen := <-rv
		if seen.err != nil {
			if !test.err {
				t.Errorf("Case #%d, unexpected error %v", ix, seen)
			}
		} else {
			if test.err {
				t.Errorf("Case #%d, expected error, saw nil", ix)
			}
		}
	}
}

func TestGetMarksResponse(t *testing.T) {
	cases := []struct {
		data  string
		marks int
		err   bool
	}{
		{"=1 3 { 13020 100 13043 95 12213 95 }", 3, false},
		{"=1 4 { 13020 100 13043 95 12213 95 }", 3, true},
	}
	for ix, test := range cases {
		c := fakeClient(test.data)
		rv := make(chan getMarksResponse)
		c.asyncMap[1] = getMarksCallback(rv)
		go c.receiveLoop()
		seen := <-rv
		if len(seen.marks) != test.marks {
			t.Errorf("Case #%d, unxpected number of marks, saw %d, expected %d", ix, len(seen.marks), test.marks)
		}
		if seen.err != nil {
			if !test.err {
				t.Errorf("Case #%d, unexpected error %v", ix, seen)
			}
		} else {
			if test.err {
				t.Errorf("Case #%d, expected error, saw nil", ix)
			}
		}
	}

}
