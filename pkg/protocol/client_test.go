package protocol
// Various client implementation tests

import (
	"testing"

	"bytes"
	"strings"

	"github.com/sirupsen/logrus"
)



func TestReadID(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	
	td := []struct{
		data string
		expected uint32
	}{
		{"2 ", 2}, {"2 22", 2}, {"22 ", 22}, {"012", 12},
		{"0 ", 0}, {"990099 ", 990099},
	}

	for ix, d := range td {
		seen := readID(strings.NewReader(d.data))
		expected := d.expected
		if seen != expected {
			t.Errorf("Case #%d, saw %d, expected %d", ix, seen, expected)
		}
	}
}

// Create a fake client, with specific data to read
func fakeClient(data string) *KomClient {
	return &KomClient{
		socket: bytes.NewBufferString(data),
		asyncMap: make(map[uint32]Callback),
		shutdown: make(chan struct{}),
	}
}

func TestReadOKAndError(t *testing.T) {
	td := []struct{
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
