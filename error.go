package pdf

import (
	"bytes"
	"fmt"
	"runtime"
)

type Error struct {
	error

	message string

	hasLocation bool
	file        string
	line        int
}

func (err Error) Error() string {
	buf := &bytes.Buffer{}

	if err.hasLocation {
		fmt.Fprintf(buf, "%v:%v: ", err.file, err.line)
	}

	fmt.Fprintf(buf, "%v", err.message)

	if err.error != nil {
		fmt.Fprintf(buf, ": %v", err.error)
	}

	// return fmt.Sprintf("%v:%v: %v: %v", err.file, err.line, err.message, err.error)
	return buf.String()
}

func (err Error) String() string {
	return err.Error()
}

func (err *Error) setLocation() {
	if _, file, line, ok := runtime.Caller(2); ok {
		err.hasLocation = true
		err.file = file
		err.line = line
	}
}

func newErr(message string) error {
	err := &Error{
		message: message,
	}
	err.setLocation()
	return err
}

func newErrf(format string, args ...interface{}) error {
	err := &Error{
		message: fmt.Sprintf(format, args...),
	}
	err.setLocation()
	return err
}

func maskErr(err error) error {
	if err == nil {
		return nil
	}

	masked := &Error{
		message: err.Error(),
	}

	masked.setLocation()

	return masked
}

func pushErr(err error, message string) error {
	pushed := Error{
		error:   err,
		message: message,
	}

	pushed.setLocation()

	return pushed
}

func pushErrf(err error, format string, args ...interface{}) error {
	pushed := Error{
		error:   err,
		message: fmt.Sprintf(format, args...),
	}

	pushed.setLocation()

	return pushed
}
