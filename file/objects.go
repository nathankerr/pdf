package file

import (
	"io"
)

/* Object can be one of the basic PDF types:
- Boolean §7.3.2
- Integer §7.3.3
- Real §7.3.3
- String §7.3.4
- Name §7.3.5
- Array §7.3.6
- Dictionary §7.3.7
- Stream §7.3.8
- Null §7.3.9
*/
type Object interface {
	io.WriterTo
}

type Boolean bool
type Integer int
type Real float64
type String []byte
type Name string
type Array []Object
type Dictionary map[Name]Object
type Stream struct {
	Dictionary
	Stream []byte
}
type Null struct{} // value here does not mean anything
type ObjectReference struct {
	ObjectNumber     uint64
	GenerationNumber uint64
}

type IndirectObject struct {
	ObjectNumber     uint64
	GenerationNumber uint64
	Object
}
