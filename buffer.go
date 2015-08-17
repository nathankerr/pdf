package pdf

import (
	"bytes"
	"fmt"
	"io"
)

// stores errors until buffer.WriteTo
type buffer struct {
	b   *bytes.Buffer
	err error
}

func (b *buffer) WriteTo(w io.Writer) (int64, error) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return 0, b.err
	}

	return b.b.WriteTo(w)
}

func (b *buffer) WriteString(s string) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return
	}

	_, b.err = b.b.WriteString(s)
}

func (b *buffer) Write(p []byte) (int, error) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return 0, b.err
	}

	var n int
	n, b.err = b.b.Write(p)
	return n, b.err
}

func (b *buffer) Printf(format string, a ...interface{}) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return
	}

	_, b.err = fmt.Fprintf(b.b, format, a...)
}

func (b *buffer) WriteByte(c byte) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return
	}

	b.err = b.b.WriteByte(c)
}

func (b *buffer) Truncate(n int) {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	if b.err != nil {
		return
	}

	b.b.Truncate(n)
}

func (b *buffer) Len() int {
	if b.b == nil {
		b.b = &bytes.Buffer{}
	}

	return b.b.Len()
}
