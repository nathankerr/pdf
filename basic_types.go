package pdf

// Specific basic object types are referred to as objects
// Boolean, Integer, Real, String, Name, Array, Dictionary,
// Stream, Null, IndirectObject
type Object interface{}

// 7.3.2 Boolean Objects
type Boolean bool

// 7.3.3 Numeric Objects
type Integer int
type Real float64

// 7.3.4 String Objects
type String []byte

// 7.3.5 Name Objects
// Should really be a []byte, but needs to be used as
// a dictionary's map key
type Name string

// 7.3.6 Array Objects
type Array []Object

// 7.3.7 Dictionary Objects
type Dictionary map[Name]Object

// 7.3.8 Stream Objects
type Stream struct {
	Dict Dictionary
	Data []byte
}

// 7.3.9 Null Object
// the type of Null does not matter
// it only matters that Null is a distinct type
type Null bool

// 7.3.10 Indirect Objects
type IndirectObject struct {
	ObjectNumber uint
	GenerationNumber uint
	Object Object
}