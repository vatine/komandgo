package protocol

// Various client implementation tests

import (
	"testing"

	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/vatine/komandgo/pkg/types"
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

func TestGetTime(t *testing.T) {
	cases := []struct {
		response string
		expected time.Time
	}{
		{"=1 23 47 19 17 6 97 4 197 1", time.Date(1997, time.June, 17, 19, 47, 23, 0, time.UTC)},
	}

	for ix, c := range cases {
		cl := fakeClient(c.response)
		rv := make(chan time.Time)
		cl.asyncMap[1] = timeResponseCallback(rv)
		go cl.receiveLoop()
		seen := <-rv
		if !seen.Equal(c.expected) {
			t.Errorf("Case %d, saw %s, expected %s", ix, seen, c.expected)
		}
	}
}

func TestGetPersonStat(t *testing.T) {
}

func cmpConfZInfo(saw, want types.ConfZInfo, t *testing.T) bool {
	ok := true
	if saw.Name != want.Name {
		t.Errorf("saw name «%s», want «%s»", saw.Name, want.Name)
		ok = false
	}

	if saw.No != want.No {
		t.Errorf("saw confNo %d, want %d", saw.No, want.No)
	}

	return ok
}

func TestReZLookup(t *testing.T) {
	cases := []struct {
		response string
		expected zConfArrayResponse
	}{
		{
			"=1 2 { 15HTest Conference 0000 10 21HTrains (-) Discussion 0000 11 }",
			zConfArrayResponse{
				confs: []types.ConfZInfo{
					types.ConfZInfo{
						Name: "Test Conference",
						No:   10,
						Type: types.ConfType{},
					},
					types.ConfZInfo{
						Name: "Trains (-) Discussion",
						No:   11,
						Type: types.ConfType{},
					},
				},
				err: nil,
			},
		},
		{
			"=1 4 { 15HTest Conference 0000 10 11HDavid Byers 1001 6 21HTrains (-) Discussion 0000 11 4HJohn 1001 9 }",
			zConfArrayResponse{
				confs: []types.ConfZInfo{
					types.ConfZInfo{
						Name: "Test Conference",
						No:   10,
						Type: types.ConfType{},
					},
					types.ConfZInfo{
						Name: "David Byers",
						No:   6,
						Type: types.ConfType{
							RdProt:    true,
							LetterBox: true,
						},
					},
					types.ConfZInfo{
						Name: "Trains (-) Discussion",
						No:   11,
						Type: types.ConfType{},
					},
					types.ConfZInfo{
						Name: "John",
						No:   9,
						Type: types.ConfType{
							RdProt:    true,
							LetterBox: true,
						},
					},
				},
				err: nil,
			},
		},
	}

	for ix, c := range cases {
		fmt.Printf("z-conf case %d\n", ix)
		cl := fakeClient(c.response)
		rv := make(chan zConfArrayResponse)
		cl.asyncMap[1] = zConfArrayResponseCallback(rv)
		go cl.receiveLoop()

		seen := <-rv
		close(cl.shutdown)

		if len(seen.confs) != len(c.expected.confs) {
			t.Errorf("Case #%d, conf array length mismatch", ix)
			continue
		}

		for cIx, got := range seen.confs {
			want := c.expected.confs[cIx]
			if !cmpConfZInfo(got, want, t) {
				t.Errorf("Case #%d, conf %d mismatch, saw %v, want %v", ix, cIx, got, want)
			}
		}
	}

}

func cmpUconf(a, b types.UConference, t *testing.T) bool {
	if a.Name != b.Name {
		t.Errorf("  name differs, saw «%s», want «%s»", a.Name, b.Name)
		return false
	}

	if a.Type.Repr() != b.Type.Repr() {
		t.Errorf("  Type differs, saw %+v, want %+v", a.Type, b.Type)
		return false
	}

	if a.HighestLocalNo != b.HighestLocalNo {
		return false
	}

	if a.Nice != b.Nice {
		return false
	}

	return true
}

func TestGetUConfStat(t *testing.T) {
	cases := []struct {
		response string
		want     types.UConference
	}{
		{
			"=1 8HTestconf 00001000 6 77",
			types.UConference{
				Name:           "Testconf",
				Type:           types.ExtendedConfType{AllowAnonymous: true},
				HighestLocalNo: 6,
				Nice:           77,
			},
		},
		{
			"=1 11HDavid Byers 11111000 0 77",
			types.UConference{
				Name: "David Byers",
				Type: types.ExtendedConfType{
					RdProt:         true,
					Original:       true,
					Secret:         true,
					LetterBox:      true,
					AllowAnonymous: true,
				},
				HighestLocalNo: 0,
				Nice:           77,
			},
		},
	}

	for ix, c := range cases {
		cl := fakeClient(c.response)
		rv := make(chan uConfResponse)
		cl.asyncMap[1] = uConfResponseCallback(rv)
		go cl.receiveLoop()
		seen := <-rv
		if !cmpUconf(seen.uConf, c.want, t) {
			t.Errorf("Case #%d, unexpected uconference, saw %+v, want %+v", ix, seen.uConf, c.want)
		}
	}
}
