package file

import (
	"io"
)

// Object can be one of the basic PDF types:
// - Boolean §7.3.2
// - Integer §7.3.3
// - Real §7.3.3
// - String §7.3.4
// - Name §7.3.5
// - Array §7.3.6
// - Dictionary §7.3.7
// - Stream §7.3.8
// - Null §7.3.9
type Object interface {
	io.WriterTo
}

// Boolean objects represent the logical values of true and false.
// - §7.3.2
type Boolean bool

// Integer objects represent mathematical integers.
// - §7.3.3
type Integer int

// Real objects represent mathematical real numbers.
// - §7.3.3
type Real float64

// A String object consists of zero or more bytes.
// - §7.3.4
type String []byte

// A Name object is an atomic symbol uniquely defined by a sequence of
// any characters (8-bit values) except null (character code 0)
// - §7.3.5
type Name string

// An Array object is a one-dimensional collection of objects
// arranged sequentially.
// - §7.3.6
type Array []Object

// A Dictionary object is an associative table mapping Names to Objects.
// - §7.3.7
type Dictionary map[Name]Object

// A Stream object is a sequence of bytes.
// - §7.3.8
type Stream struct {
	Dictionary
	Stream []byte
}

// The Null object has a type and value that are unequal to any other object.
// - §7.3.9
type Null struct{}

// An ObjectReference references a specific Object with the exact
// ObjectNumber and GenerationNumbers specified.
// Indirectly defined in
type ObjectReference struct {
	ObjectNumber     uint64 // positive integer
	GenerationNumber uint64 // non-negative integer
}

// An IndirectObject gives an Object an ObjectReference by which
// other Objects can refer to it.
// - §7.3.10
type IndirectObject struct {
	ObjectReference
	Object
}
