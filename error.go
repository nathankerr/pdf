package pdf

import (
	"bytes"
	"fmt"
	"runtime"
)

// an error stack
type pdfError struct {
	error

	message string

	hasLocation bool
	file        string
	line        int
}

func (err pdfError) Error() string {
	buf := &bytes.Buffer{}

	if err.hasLocation {
		fmt.Fprintf(buf, "%v:%v: ", err.file, err.line)
	}

	fmt.Fprintf(buf, "%v", err.message)

	if err.error != nil {
		fmt.Fprintf(buf, ": %v", err.error)
	}

	return buf.String()
}

func (err pdfError) String() string {
	return err.Error()
}

func (err *pdfError) setLocation() {
	if _, file, line, ok := runtime.Caller(2); ok {
		err.hasLocation = true
		err.file = file
		err.line = line
	}
}

func newErr(message string) error {
	err := &pdfError{
		message: message,
	}
	err.setLocation()
	return err
}

func newErrf(format string, args ...interface{}) error {
	err := &pdfError{
		message: fmt.Sprintf(format, args...),
	}
	err.setLocation()
	return err
}

func maskErr(err error) error {
	if err == nil {
		return nil
	}

	masked := &pdfError{
		message: err.Error(),
	}

	masked.setLocation()

	return masked
}

func pushErr(err error, message string) error {
	pushed := pdfError{
		error:   err,
		message: message,
	}

	pushed.setLocation()

	return pushed
}

func pushErrf(err error, format string, args ...interface{}) error {
	pushed := pdfError{
		error:   err,
		message: fmt.Sprintf(format, args...),
	}

	pushed.setLocation()

	return pushed
}
