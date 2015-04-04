package pdf

import (
	"bytes"
	"fmt"
	"io"
)

// WriteTo serializes the Boolean according to the rules in
// §7.3.2
func (b Boolean) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	if b {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}

	return buf.WriteTo(w)
}

// WriteTo serializes the Integer according to the rules in
// §7.3.3
func (i Integer) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d", int(i))

	return buf.WriteTo(w)
}

// WriteTo serializes the Real according to the rules in
// §7.3.3
func (r Real) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%v", float64(r))

	return buf.WriteTo(w)
}

// WriteTo serializes the String according to the rules in
// §7.3.4
func (s String) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteByte('(')
	for _, b := range []byte(s) {
		switch b {
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

// WriteTo serializes the Name according to the rules in
// §7.3.5
func (n Name) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "/%s", n)

	return buf.WriteTo(w)
}

// WriteTo serializes the Array according to the rules in
// §7.3.6
func (a Array) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteByte('[')
	for _, obj := range a {
		obj.writeTo(buf)
		buf.WriteByte(' ')
	}
	if len(a) != 0 {
		buf.Truncate(buf.Len() - 1)
	}
	buf.WriteByte(']')

	return buf.WriteTo(w)
}

// WriteTo serializes the Dictionary according to the rules in
// §7.3.6
func (d Dictionary) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteString("<<")
	for name, obj := range d {
		name.writeTo(buf)
		buf.WriteByte(' ')
		obj.writeTo(buf)
	}
	buf.WriteString(">>")

	return buf.WriteTo(w)
}

// WriteTo serializes the Stream according to the rules in
// §7.3.8
func (s Stream) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	// update the dictionary
	if s.Dictionary == nil {
		s.Dictionary = Dictionary{}
	}
	s.Dictionary[Name("Length")] = Integer(len(s.Stream))

	s.Dictionary.writeTo(buf)

	fmt.Fprintf(buf, "\nstream\n")
	buf.Write(s.Stream)
	fmt.Fprintf(buf, "\nendstream")

	return buf.WriteTo(w)
}

// WriteTo serializes Null according to the rules in
// §7.3.9
func (null Null) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	buf.WriteString("null")

	return buf.WriteTo(w)
}

// WriteTo serializes the ObjectReference according to the rules in
// §7.3.10
func (objref ObjectReference) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "%d %d R", objref.ObjectNumber, objref.GenerationNumber)

	return buf.WriteTo(w)
}

// WriteTo serializes the IndirectObject according to the rules in
// §7.3.10
func (inobj IndirectObject) writeTo(w io.Writer) (int64, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%d %d obj\n", inobj.ObjectNumber, inobj.GenerationNumber)
	inobj.Object.writeTo(buf)
	fmt.Fprintf(buf, "\nendobj")
	return buf.WriteTo(w)
}
