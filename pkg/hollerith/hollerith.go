// A package for reading and writing Hollerith strings
package hollerith

import (
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/vatine/komandgo/pkg/utils"
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

func Fprintf(sink io.Writer, f string, args ...interface{}) (int, error) {
	i := fmt.Sprintf(f, args...)
	return Fprint(sink, i)
}

func Sprint(s interface{}) string {
	i := fmt.Sprint(s)
	return fmt.Sprintf("%dH%s", len(i), i)
}

func Scan(source io.Reader) (string, error) {
	l := 0

	for done := false; !done; _ = done {
		b, e := utils.ReadByte(source)

		if e != nil {
			return "", e
		}

		done = (b == 'H')
		if !done {
			p := strings.IndexByte("0123456789", b)
			log.WithFields(log.Fields{
				"p": p,
				"b": b,
			}).Debug("reading length")
			if p >= 0 {
				l = l*10 + p
			} else {
				log.WithFields(log.Fields{
					"p": p,
					"b": b,
				}).Error("unexpected character")
				return "", HollerithError("Unexpected length character")
			}
		}
	}

	rv := []byte{}
	for c := 0; c < l; c++ {
		b, e := utils.ReadByte(source)
		if e != nil {
			return "", e
		}
		rv = append(rv, b)
	}

	return string(rv), nil
}
