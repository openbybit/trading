package errors

import (
	"fmt"
	"io"
)

// Errors collects multiple errors as a slice
type Errors []error

// Format implements io.Formatter
// nolint errcheck // not handling errors during format
func (ep *Errors) Format(f fmt.State, verb rune) {
	io.WriteString(f, ep.Error())
	f.Write([]byte{'\n'})
	if ep.Len() == 0 {
		return
	}
	for _, err := range *ep {
		if verb == 's' {
			fmt.Fprintf(f, "%s\n", err)
			continue
		}
		if verb != 'v' {
			fmt.Fprintf(f, "%%!%c(%s)\n", verb, err.Error())
			continue
		}
		if f.Flag('+') {
			fmt.Fprintf(f, "%[1]T: %+[1]v\n", err)
		} else {
			fmt.Fprintf(f, "%[1]T: %[1]v\n", err)
		}
	}
}

// Error note that event the slice is empty or nil, the value still implements error
// interface. Use Err() to handle this.
func (ep *Errors) Error() string {
	return fmt.Sprintf("<%d errors>", ep.Len())
}

// Collect appends err into *ep.
func (ep *Errors) Collect(err error) {
	if err != nil {
		*ep = append(*ep, err)
	}
}

// Len returns length of *ep, nil is handled.
func (ep *Errors) Len() int {
	if ep == nil {
		return 0
	}
	return len(*ep)
}

// Err returns nil if ep is nil or empty slice.
func (ep *Errors) Err() error {
	if ep.Len() == 0 {
		return nil
	}

	return ep
}
