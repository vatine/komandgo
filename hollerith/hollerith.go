// A package for reading and writing Hollerith strings
package hollerith

import (
	"fmt"
	"io"
)

// Writes a string to an io.Writer, simply returns any underlying error as
// and when they occur.
func Fprint(sink io.Writer, s fmt.Stringer) int, error {
	i := s.String()
	return fmt.Fprintf(sink, "%dH%s", len(i), i)
}

func Fprintf(sink io.Writer, fmt string, args interface{}...) int, error {
	i := fmt.Sprintf(fmt, args...)
	return Fprintf(sink, i)
}
