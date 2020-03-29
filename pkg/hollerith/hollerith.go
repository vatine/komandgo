// A package for reading and writing Hollerith strings
package hollerith

import (
	"fmt"
	"io"
	"strings"
)

type HollerithError string

func (e HollerithError) Error() string {
	return string(e)
}

// Writes a string to an io.Writer, simply returns any underlying error as
// and when they occur.
func Fprint(sink io.Writer, s interface{}) (int, error) {
	i := fmt.Sprint(s)
	return fmt.Fprintf(sink, "%dH%s", len(i), i)
}

func Fprintf(sink io.Writer, f string, args... interface{}) (int, error) {
	i := fmt.Sprintf(f, args...)
	return Fprint(sink, i)
}

func Scan(source io.RuneReader) (string, error) {
	l := 0

	for done := false; !done; _ = done {
		r, _, e := source.ReadRune()

		if e != nil {
			return "", e
		}

		done = (r == 'H')
		if !done {
			p := strings.IndexRune("0123456789", r)
			fmt.Printf("p = %d, r = %c\n", p, r)
			if p >= 0 {
				l = l * 10 + p
			} else {
				return "", HollerithError("Unexpected length character")
			}
		}
	}

	rv := []rune{}
	for c := 0; c < l; _ = c {
		r, cnt, e := source.ReadRune()
		if e != nil {
			return "", e
		}
		c += cnt
		rv = append(rv, r)
	}

	return string(rv), nil
}
