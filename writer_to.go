package pdf

import (
	"io"
)

// WriteTo serializes the Boolean according to the rules in
// §7.3.2
func (b Boolean) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

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
	buf := &buffer{}

	buf.Printf("%d", int(i))

	return buf.WriteTo(w)
}

// WriteTo serializes the Real according to the rules in
// §7.3.3
func (r Real) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

	buf.Printf("%v", float64(r))

	return buf.WriteTo(w)
}

// WriteTo serializes the String according to the rules in
// §7.3.4
func (s String) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

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
	buf := &buffer{}

	buf.Printf("/%s", n)

	return buf.WriteTo(w)
}

// WriteTo serializes the Array according to the rules in
// §7.3.6
func (a Array) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

	buf.WriteByte('[')
	for _, obj := range a {
		n, err := obj.writeTo(buf)
		if err != nil {
			return n, err
		}
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
	buf := &buffer{}

	buf.WriteString("<<")
	for name, obj := range d {
		n, err := name.writeTo(buf)
		if err != nil {
			return n, err
		}
		buf.WriteByte(' ')
		n, err = obj.writeTo(buf)
		if err != nil {
			return n, err
		}
	}
	buf.WriteString(">>")

	return buf.WriteTo(w)
}

// WriteTo serializes the Stream according to the rules in
// §7.3.8
func (s Stream) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

	// update the dictionary
	if s.Dictionary == nil {
		s.Dictionary = Dictionary{}
	}
	s.Dictionary[Name("Length")] = Integer(len(s.Stream))

	n, err := s.Dictionary.writeTo(buf)
	if err != nil {
		return n, err
	}

	buf.Printf("\nstream\n")
	nint, err := buf.Write(s.Stream)
	if err != nil {
		return int64(nint), err
	}
	buf.Printf("\nendstream")

	return buf.WriteTo(w)
}

// WriteTo serializes Null according to the rules in
// §7.3.9
func (null Null) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

	buf.WriteString("null")

	return buf.WriteTo(w)
}

// WriteTo serializes the ObjectReference according to the rules in
// §7.3.10
func (objref ObjectReference) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}

	buf.Printf("%d %d R", objref.ObjectNumber, objref.GenerationNumber)

	return buf.WriteTo(w)
}

// WriteTo serializes the IndirectObject according to the rules in
// §7.3.10
func (inobj IndirectObject) writeTo(w io.Writer) (int64, error) {
	buf := &buffer{}
	buf.Printf("%d %d obj\n", inobj.ObjectNumber, inobj.GenerationNumber)

	n, err := inobj.Object.writeTo(buf)
	if err != nil {
		return n, err
	}

	buf.Printf("\nendobj")
	return buf.WriteTo(w)
}
