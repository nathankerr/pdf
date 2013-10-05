package file

import (
	"errors"
	"strconv"
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
type Object interface{}
type String []byte

type IndirectObject struct {
	ObjectNumber     uint64
	GenerationNumber uint64
	Object
}

func ParseIndirectObject(slice []byte) (*IndirectObject, error) {
	io := new(IndirectObject)

	start := 0
	var err error

	// Object Number
	token, n := nextToken(slice[start:])
	start += n
	io.ObjectNumber, err = strconv.ParseUint(string(token), 10, 64)
	if err != nil {
		return nil, err
	}

	// Generation Number
	token, n = nextToken(slice[start:])
	start += n
	io.GenerationNumber, err = strconv.ParseUint(string(token), 10, 64)
	if err != nil {
		return nil, err
	}

	// "obj"
	n, ok := match(slice[start:], "obj")
	if !ok {
		return nil, errors.New("could not find 'obj'")
	}
	start += n

	// the object
	if n, ok := nextNonWhitespace(slice[start:]); ok {
		start += n
	}

	// determine the object type
	// except for Stream §7.3.8
	// streams start as dictionaries
	switch slice[start] {
	case 't', 'f':
		// Boolean §7.3.2
		println("Boolean")
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-':
		// Integer §7.3.3
		// Real §7.3.3
		println("Numeric")
	case '(':
		// String §7.3.4
		println("Literal String")
		io.Object, err = ParseLiteralString(slice[start:])
	case '/':
		// Name §7.3.5
		println("Name")
	case '[':
		// Array §7.3.6
		println("Array")
	case '<':
		if slice[start+1] == '<' {
			// Dictionary §7.3.7
			println("Dictionary")
		} else {
			// String §7.3.4
			println("Hexadecimal String")
		}
	case 'n':
		// Null §7.3.9
		println("Null")
	default:
		panic(string(slice[start]))
	}

	// switch object.(type) {
	// case Dictionary:
	// 	// check to see if it is really a stream
	// }

	return io, nil
}

// for tokenized things, returns the next token
func nextToken(slice []byte) ([]byte, int) {
	// whitespace:
	// null, tab, line feed, form feed, carriage return, or space
	// §7.2.2 Table 1

	// delimiters:
	// (, ), <, >, [, ], {, }, /, %
	// §7.2.2 Table 2

	var begin, end int

	begin, ok := nextNonWhitespace(slice)
	if !ok {
		begin = 0
	}

	for end = begin; end < len(slice); end++ {
		switch slice[end] {
		case 0, 9, 10, 12, 13, 32, // whitespace
			40, 41, 60, 62, 91, 93, 123, 125, 47, 37: // delimiters
			return slice[begin:end], end - begin + 1
		}
	}

	return slice[begin:], len(slice[begin:])
}

func nextNonWhitespace(slice []byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		switch slice[i] {
		case 0, 9, 10, 12, 13, 32: // whitespace
		default:
			return i, true
		}
	}
	return -1, false
}

func match(slice []byte, toMatch string) (int, bool) {
	start, ok := nextNonWhitespace(slice)
	if !ok {
		return -1, false
	}

	for i := 0; i < len(toMatch); i++ {
		if slice[start+i] != toMatch[i] {
			return -1, false
		}
	}

	return start + len(toMatch), true
}

func index(slice []byte, toFind byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == toFind {
			return i, true
		}
	}
	return -1, false
}

func ParseLiteralString(slice []byte) (String, error) {
	if slice[0] != '(' {
		return nil, errors.New("not a literal string")
	}

	endParen, ok := index(slice, ')')
	if !ok {
		return nil, errors.New("couldn't find end of string")
	}

	return slice[1:endParen], nil
}
