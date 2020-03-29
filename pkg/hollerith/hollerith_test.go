package hollerith

import (
	"bytes"
	"testing"
)

func TestBasicPrinting (t *testing.T) {
	sink := bytes.NewBuffer(nil)

	testData := []struct {
		data     interface{}
		expected string
	}{{1, "1H1"}, {"1", "1H1"}, {"1H1", "3H1H1"}, {"räksmörgås", "13Hräksmörgås"}}

	for ix, td := range testData {
		sink.Reset()
		_, e := Fprint(sink, td.data)
		switch {
		case e != nil:
			t.Errorf("Hollerith-printing %s unexpectedly caused error %s at ix %d", td.data, e, ix)
		case td.expected != sink.String():
			t.Errorf("saw %s, expected %s at ix %d", sink.String(), td.expected, ix)
		default:
			continue
		}
	}
		
}

func TestFormattedPrinting (t *testing.T) {
	sink := bytes.NewBuffer(nil)

	testData := []struct {
		f        string
		data     []interface{}
		expected string
	} {
		{"%d%d%d", []interface{}{1, 2, 3}, "3H123"},
		{"%d%s", []interface{}{1, "23"}, "3H123"},
		{"%d", []interface{}{-4711}, "5H-4711"},
		{"%s %s", []interface{}{"Hello", "world!"}, "12HHello world!"},
		{"%s%s%s", []interface{}{"räk", "smör", "gås"}, "13Hräksmörgås"},
	}

	for ix, td := range testData {
		sink.Reset()
		_, e := Fprintf(sink, td.f, td.data...)
		switch {
		case e != nil:
			t.Errorf("Error signalled, %s, ix %d", e, ix)
		case td.expected != sink.String():
			t.Errorf("Expected %s, saw %s", td.expected, sink.String())
		}
	}
}

func TestScanning(t *testing.T) {
	testData := []struct{
		source   string
		expected string
		err      bool
	}{
		{"13Hräksmörgås", "räksmörgås", false},
		{"130Hräksmörgås", "räksmörgås", true},
		{"3H1H1", "1H1", false},
		{"13%Hhalvah", "", true},
	}

	for ix, td := range testData {
		source := bytes.NewBufferString(td.source)
		seen, e := Scan(source)

		switch {
		case (e != nil) && !td.err:
			t.Errorf("Saw unexpected error %s, ix %d", e, ix)
		case td.err:
			if e == nil {
				t.Errorf("Did not see expected error, ix %d", ix)
			}
		case seen != td.expected:
			t.Errorf("Saw %s, expected %s, ix %d", seen, td.expected, ix)
		}
		
	}
}
