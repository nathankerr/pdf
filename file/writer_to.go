package file

import (
	"bytes"
	"fmt"
	"io"
)

// §7.3.2
func (b Boolean) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	if b {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}

	return buf.WriteTo(w)
}

// §7.3.3
func (i Integer) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d", int(i))

	return buf.WriteTo(w)
}

// §7.3.3
func (r Real) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d", float64(r))

	return buf.WriteTo(w)
}

// §7.3.4
func (s String) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteByte('(')
	for _, b := range []byte(s) {
		switch b {
		case '\\':
			buf.WriteString("\\\\")
		case '(':
			buf.WriteString("\\(")
		case ')':
			buf.WriteString("\\)")
		default:
			buf.WriteByte(b)
		}
	}
	buf.WriteByte(')')

	return buf.WriteTo(w)
}

// §7.3.5
func (n Name) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "/%s", n)

	return buf.WriteTo(w)
}

// §7.3.6
func (a Array) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteByte('[')
	for _, obj := range a {
		obj.WriteTo(buf)
		buf.WriteByte(' ')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteByte(']')

	return buf.WriteTo(w)
}

// §7.3.6
func (d Dictionary) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteString("<<")
	for name, obj := range d {
		name.WriteTo(buf)
		buf.WriteByte(' ')
		obj.WriteTo(buf)
	}
	buf.WriteString(">>")

	return buf.WriteTo(w)
}

// §7.3.8
// should always be called from IndirectObject.WriteTo
func (s Stream) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	s.Dictionary.WriteTo(buf)

	fmt.Fprintf(buf, "\nstream\n")
	buf.Write(s.Stream)
	fmt.Fprintf(buf, "\nendstream")

	return buf.WriteTo(w)
}

// §7.3.9
func (null Null) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteString("null")

	return buf.WriteTo(w)
}

// §7.3.10
func (objref ObjectReference) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d %d R", objref.ObjectNumber, objref.GenerationNumber)

	return buf.WriteTo(w)
}

// §7.3.10
func (inobj IndirectObject) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%d %d obj\n", inobj.ObjectNumber, inobj.GenerationNumber)
	inobj.Object.WriteTo(buf)
	fmt.Fprintf(buf, "\nendobj")
	return buf.WriteTo(w)
}
