package util

import (
	"fmt"
	"strings"
)

// MultiError combines a number of errors into a single error value.
type MultiError []error

func (me *MultiError) FromChan(errs chan error) *MultiError {
	for err := range errs {
		(*me) = append((*me), err)
	}
	return me
}

func (me MultiError) IsEmpty() bool {
	return len(me) == 0
}

func (me MultiError) Error() string {
	if len(me) == 1 {
		return me[0].Error()
	}
	var b strings.Builder
	b.WriteString("[\n")
	for ii, err := range me {
		fmt.Fprintf(&b, "\t%d: %s\n", ii+1, err)
	}
	b.WriteString("]\n")
	return b.String()
}
