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

type parseFn func(slice []byte) (Object, int, error)

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
		io.Object, n, err = ParseLiteralString(slice[start:])
		start += n
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

func isDelimiter(char byte) bool {
	switch char {
	case 40, 41, 60, 62, 91, 93, 123, 125, 47, 37: // delimiters
		return true
	}
	return false
}

func isWhitespace(char byte) bool {
	switch char {
	case 0, 9, 10, 12, 13, 32: // whitespace
		return true
	}
	return false
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

func nextWhitespace(slice []byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		switch slice[i] {
		case 0, 9, 10, 12, 13, 32: // whitespace
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

func ParseLiteralString(slice []byte) (Object, int, error) {
	if slice[0] != '(' {
		return nil, 0, errors.New("not a literal string")
	}

	parens := 0
	decoded := make([]byte, len(slice))
	decodedIndex := 0
	for i := 0; i < len(slice); i++ {
		include := true
		switch slice[i] {
		case '(':
			if parens == 0 {
				include = false
			}
			parens++
		case ')':
			parens--
			if parens == 0 {
				return String(decoded[:decodedIndex]), i + 1, nil
			} else {
				include = true
			}
		case '\n':
			if slice[i-1] == '\\' {
				decodedIndex--
				include = false
			}
		}

		if include {
			decoded[decodedIndex] = slice[i]
			decodedIndex++
		}
	}

	return nil, 0, errors.New("couldn't find end of string")
}

// returned int is the length of slice consumed
func ParseDictionary(slice []byte) (Object, int, error) {
	if slice[0] != '<' && slice[1] != '<' {
		return nil, 0, errors.New("not a dictionary")
	}

	dict := make(Dictionary)

	i := 2
	for i < len(slice) {
		// skip whitespace
		n, ok := nextNonWhitespace(slice[i:])
		if !ok {
			return nil, 0, errors.New("expected a non-whitespace char")
		}
		i += n

		// check to see if end
		if slice[i] == '>' && slice[i+1] == '>' {
			i += 2
			break
		}

		// get the key
		name, n, err := ParseName(slice[i:])
		if err != nil {
			return nil, 0, err
		}
		i += n

		key, ok := name.(Name)
		if !ok {
			return nil, 0, errors.New("unable to cast Name")
		}

		// get the value
		value, n, err := ParseObject(slice[i:])
		if err != nil {
			return nil, 0, err
		}
		i += n

		// set the key/value pair
		dict[key] = value
	}

	return dict, i, nil
}

func ParseName(slice []byte) (Object, int, error) {
	if slice[0] != '/' {
		return Name(""), 0, errors.New("not a name")
	}

	name := make([]byte, 0, len(slice))

	i := 1
	for i < len(slice) {
		if isDelimiter(slice[i]) || isWhitespace(slice[i]) {
			break
		}

		switch slice[i] {
		case '#':
			char, err := strconv.ParseUint(string(slice[i+1:i+3]), 16, 8)
			if err != nil {
				return Name(""), 0, err
			}
			name = append(name, byte(char))
			i += 2
		default:
			name = append(name, slice[i])
		}
		i++
	}

	return Name(name), i, nil
}

func ParseBoolean(slice []byte) (Object, int, error) {
	n, ok := match(slice, "true")
	if ok {
		return Boolean(true), n, nil
	}

	n, ok = match(slice, "false")
	if ok {
		return Boolean(false), n, nil
	}

	return Boolean(false), 0, errors.New("not a boolean")
}

// returns Integer when integer, Real when real
func ParseNumeric(slice []byte) (Object, int, error) {
	token, n := nextToken(slice)

	isInteger := true
	for _, char := range token {
		if char == '.' {
			isInteger = false
			break
		}
	}

	if isInteger {
		integer, err := strconv.ParseInt(string(token), 10, 0)
		if err != nil {
			return Integer(0), 0, err
		}

		return Integer(integer), n, nil
	}

	real, err := strconv.ParseFloat(string(token), 64)
	if err != nil {
		return Real(0), 0, err
	}

	return Real(real), n, nil
}

func ParseHexadecimalString(slice []byte) (Object, int, error) {
	hex := make(String, 0, int(len(slice)/2))

	if slice[0] != '<' {
		return hex, 0, errors.New("not a hexadecimal string")
	}

	i := 1
	for i < len(slice) {
		if slice[i] == '>' {
			i++
			break
		}

		if isHexDigit(slice[i]) && isHexDigit(slice[i+1]) {
			b, err := strconv.ParseUint(string(slice[i:i+2]), 16, 8)
			if err != nil {
				return hex, 0, err
			}
			hex = append(hex, byte(b))
			i += 2
			continue
		}

		if isHexDigit(slice[i]) && slice[i+1] == '>' {
			b, err := strconv.ParseUint(string(slice[i])+"0", 16, 8)
			if err != nil {
				return hex, 0, err
			}
			hex = append(hex, byte(b))
			i += 2
			break
		}
	}

	return hex, i, nil
}

func isHexDigit(char byte) bool {
	switch char {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'A', 'B', 'C', 'D', 'E', 'F',
		'a', 'b', 'c', 'd', 'e', 'f':
		return true
	}
	return false
}

func ParseObject(slice []byte) (Object, int, error) {
	start, ok := nextNonWhitespace(slice)
	if !ok {
		return nil, 0, errors.New("expected a non-whitespace char")
	}

	var parser parseFn

	// println("\t" + string(slice[start:]))

	// determine the object type
	// except for Stream §7.3.8
	// streams start as dictionaries
	switch slice[start] {
	case 't', 'f':
		// Boolean §7.3.2
		parser = ParseBoolean
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-':
		// Integer §7.3.3
		// Real §7.3.3
		parser = ParseNumeric
	case '(':
		// String §7.3.4
		parser = ParseLiteralString
	case '/':
		// Name §7.3.5
		parser = ParseName
	case '[':
		// Array §7.3.6
		parser = ParseArray
	case '<':
		if slice[start+1] == '<' {
			// Dictionary §7.3.7
			parser = ParseDictionary
		} else {
			// String §7.3.4
			parser = ParseHexadecimalString
		}
	case 'n':
		// Null §7.3.9
		parser = ParseNull
	default:
		panic(string(slice[start]))
	}

	object, n, err := parser(slice[start:])

	return object, start + n, err
}

func ParseArray(slice []byte) (Object, int, error) {
	array := make(Array, 0)

	if slice[0] != '[' {
		return array, 0, errors.New("not an array")
	}

	i := 1
	for i < len(slice) {
		if slice[i] == ']' {
			return array, i + 1, nil
		}

		object, n, err := ParseObject(slice[i:])
		if err != nil {
			return array, 0, err
		}
		i += n

		array = append(array, object)
	}

	return array, i, errors.New("end of array not found")
}

func ParseNull(slice []byte) (Object, int, error) {
	n, ok := match(slice, "null")
	if ok {
		return Null{}, n, nil
	}

	return nil, 0, errors.New("not a Null")
}
