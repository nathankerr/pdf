package file

import (
	"io"
)

// Object represents all of the types that can be handled
// by the file store. Those types (defined in this package) are:
//   - Boolean
//   - Integer
//   - Real
//   - String
//   - Name
//   - Array
//   - Dictionary
//   - Stream
//   - Null
type Object interface {
	thisIsABasicPDFObject()
	io.WriterTo
}

// Boolean objects represent the logical values of true and false.
// - §7.3.2
type Boolean bool

func (Boolean) thisIsABasicPDFObject() {}

// Integer objects represent mathematical integers.
// - §7.3.3
type Integer int

func (Integer) thisIsABasicPDFObject() {}

// Real objects represent mathematical real numbers.
// - §7.3.3
type Real float64

func (Real) thisIsABasicPDFObject() {}

// A String object consists of zero or more bytes.
// - §7.3.4
type String []byte

func (String) thisIsABasicPDFObject() {}

// A Name object is an atomic symbol uniquely defined by a sequence of
// any characters (8-bit values) except null (character code 0)
// - §7.3.5
type Name string

func (Name) thisIsABasicPDFObject() {}

// An Array object is a one-dimensional collection of objects
// arranged sequentially.
// - §7.3.6
type Array []Object

func (Array) thisIsABasicPDFObject() {}

// A Dictionary object is an associative table mapping Names to Objects.
// - §7.3.7
type Dictionary map[Name]Object

func (Dictionary) thisIsABasicPDFObject() {}

// A Stream object is a sequence of bytes.
// - §7.3.8
type Stream struct {
	Dictionary
	Stream []byte
}

func (Stream) thisIsABasicPDFObject() {}

// The Null object has a type and value that are unequal to any other object.
// - §7.3.9
type Null struct{}

func (Null) thisIsABasicPDFObject() {}

// An ObjectReference references a specific Object with the exact
// ObjectNumber and GenerationNumbers specified.
// Indirectly defined in
type ObjectReference struct {
	ObjectNumber     uint // positive integer
	GenerationNumber uint // non-negative integer
}

func (ObjectReference) thisIsABasicPDFObject() {}

// An IndirectObject gives an Object an ObjectReference by which
// other Objects can refer to it.
// - §7.3.10
type IndirectObject struct {
	ObjectReference
	Object
}

func (IndirectObject) thisIsABasicPDFObject() {}
