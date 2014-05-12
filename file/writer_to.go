package file

import (
	"bytes"
	"fmt"
	"io"
)

func (b Boolean) WriteTo(w io.Writer) (int64, error) {
	panic("not implemented")
}

func (i Integer) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d", int(i))

	return buf.WriteTo(w)
}

func (r Real) WriteTo(w io.Writer) (int64, error) {
	panic("not implemented")
}

func (s String) WriteTo(w io.Writer) (int64, error) {
	panic("not implemented")
}

func (n Name) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "/%s", n)

	return buf.WriteTo(w)
}

func (a Array) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "[ ")
	for _, obj := range a {
		obj.WriteTo(buf)
		fmt.Fprintf(buf, " ")
	}
	fmt.Fprintf(buf, "]")

	return buf.WriteTo(w)
}

func (d Dictionary) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "<< ")
	for name, obj := range d {
		name.WriteTo(buf)
		fmt.Fprintf(buf, " ")
		obj.WriteTo(buf)
		fmt.Fprintf(buf, "\n")
	}
	fmt.Fprintf(buf, ">>")

	return buf.WriteTo(w)
}

// should always be called from IndirectObject.WriteTo
func (s Stream) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	s.Dictionary.WriteTo(buf)

	fmt.Fprintf(buf, "\nstream\n")
	buf.Write(s.Stream)
	fmt.Fprintf(buf, "endstream")

	return buf.WriteTo(w)
}

func (null Null) WriteTo(w io.Writer) (int64, error) {
	panic("not implemented")
}

func (objref ObjectReference) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d %d R", objref.ObjectNumber, objref.GenerationNumber)

	return buf.WriteTo(w)
}

func (inobj IndirectObject) WriteTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%d %d obj\n", inobj.ObjectNumber, inobj.GenerationNumber)
	inobj.Object.WriteTo(buf)
	fmt.Fprintf(buf, "\nendobj")
	return buf.WriteTo(w)
}
