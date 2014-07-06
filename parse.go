package pdf

import (
	"errors"
	"strconv"
)

// Returns an Object and the number of bytes consumed
// if err != nil, the int is the offset in the slice
// where the error was discovered. The object will
// be returned as far as it was completed (to allow
// for inspection)
type parseFn func(slice []byte) (Object, int, error)

func parseObject(slice []byte) (Object, int, error) {
	start, ok := nextNonWhitespace(slice)
	if !ok {
		return nil, 0, errors.New("expected a non-whitespace char")
	}

	var parser parseFn
	maybeObjectReference := false
	maybeStream := false

	// determine the object type
	// except for Stream §7.3.8
	// streams start as dictionaries
	switch slice[start] {
	case 't', 'f':
		// Boolean §7.3.2
		parser = parseBoolean
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '-':
		// Integer §7.3.3
		// Real §7.3.3
		// could also be the start of an object reference
		parser = parseNumeric
		maybeObjectReference = true
	case '(':
		// String §7.3.4
		parser = parseLiteralString
	case '/':
		// Name §7.3.5
		// println("Name")
		parser = parseName
	case '[':
		// Array §7.3.6
		parser = parseArray
	case '<':
		if slice[start+1] == '<' {
			// Dictionary §7.3.7
			// println("Dictionary")
			parser = parseDictionary
			maybeStream = true
		} else {
			// String §7.3.4
			parser = parseHexadecimalString
		}
	case 'n':
		// Null §7.3.9
		parser = parseNull
	default:
		panic(string(slice[start]))
	}

	object, n, err := parser(slice[start:])

	if maybeObjectReference {
		objectref, n2, err := parseObjectReference(slice[start:])
		if err == nil {
			object = objectref
			n = n2
		}
	}

	// handle streams
	if maybeStream {
		n2, isStream := match(slice[start+n:], "stream")
		if isStream {
			n += n2

			// consume end of line (§7.3.8.1 paragraph after example)
			switch slice[start+n] {
			case 13: // carriage return
				n++
				if slice[start+n] != '\n' {
					return object, start + n + 1, errors.New("end of line marker cannot have only a carriage return")
				}
			case '\n': // new line
			default:
				return object, start + n + 1, errors.New("expected end of line marker")
			}
			n++

			dict, isDictionary := object.(Dictionary)
			if !isDictionary {
				return object, start + n, errors.New("expected a Dictionary")
			}

			streamLength := int(dict["Length"].(Integer))
			object = Stream{
				Dictionary: dict,
				Stream:     slice[start+n : start+n+streamLength],
			}
			n += streamLength

			n2, ok = match(slice[start+n:], "endstream")
			n += n2
			if !ok {
				return object, start + n, errors.New("expected 'endstream'")
			}
		}
	}

	return object, start + n, err
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
		if isWhitespace(slice[end]) || isDelimiter(slice[end]) {
			return slice[begin:end], end
		}
	}

	return slice[begin:], len(slice)
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

func isHexDigit(char byte) bool {
	switch char {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'A', 'B', 'C', 'D', 'E', 'F',
		'a', 'b', 'c', 'd', 'e', 'f':
		return true
	}
	return false
}

func nextNonWhitespace(slice []byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		if !isWhitespace(slice[i]) {
			return i, true
		}
	}
	return 0, false
}

func nextWhitespace(slice []byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		if isWhitespace(slice[i]) {
			return i, true
		}
	}
	return 0, false
}

func match(slice []byte, toMatch string) (int, bool) {
	token, n := nextToken(slice)

	if len(token) != len(toMatch) {
		return 0, false
	}

	for i, char := range token {
		if char != toMatch[i] {
			return n + i, false
		}
	}

	return n, true
}

func index(slice []byte, toFind byte) (int, bool) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == toFind {
			return i, true
		}
	}
	return 0, false
}

func parseLiteralString(slice []byte) (Object, int, error) {
	decoded := make([]byte, len(slice))
	decodedIndex := 0

	if slice[0] != '(' {
		return String(decoded[:decodedIndex]), 0, errors.New("not a literal string")
	}

	parens := 0
	i := 0
	for i < len(slice) {
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
			}
			include = true
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
		i++
	}

	return String(decoded[:decodedIndex]), i, errors.New("couldn't find end of string")
}

// returned int is the length of slice consumed
func parseDictionary(slice []byte) (Object, int, error) {
	dict := make(Dictionary)

	if slice[0] != '<' && slice[1] != '<' {
		return dict, 0, errors.New("not a dictionary")
	}

	i := 2
	for i < len(slice) {
		// skip whitespace
		n, ok := nextNonWhitespace(slice[i:])
		if !ok {
			return dict, i, errors.New("expected a non-whitespace char")
		}
		i += n

		// check to see if end
		if slice[i] == '>' && slice[i+1] == '>' {
			i += 2
			break
		}

		// get the key
		name, n, err := parseName(slice[i:])
		if err != nil {
			return dict, i + n, err
		}
		i += n

		key, ok := name.(Name)
		if !ok {
			return dict, i, errors.New("unable to cast Name")
		}

		// get the value
		value, n, err := parseObject(slice[i:])
		if err != nil {
			return dict, i, err
		}
		i += n

		// set the key/value pair
		dict[key] = value
	}

	return dict, i, nil
}

func parseName(slice []byte) (Object, int, error) {
	name := make([]byte, 0, len(slice))

	if slice[0] != '/' {
		return Name(name), 0, errors.New("not a name")
	}

	i := 1
	for i < len(slice) {
		if isDelimiter(slice[i]) || isWhitespace(slice[i]) {
			break
		}

		switch slice[i] {
		case '#':
			char, err := strconv.ParseUint(string(slice[i+1:i+3]), 16, 8)
			if err != nil {
				return Name(name), i, err
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

func parseBoolean(slice []byte) (Object, int, error) {
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
func parseNumeric(slice []byte) (Object, int, error) {
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
			return Integer(integer), n, err
		}

		return Integer(integer), n, nil
	}

	real, err := strconv.ParseFloat(string(token), 64)
	if err != nil {
		return Real(0), n, err
	}

	return Real(real), n, nil
}

func parseHexadecimalString(slice []byte) (Object, int, error) {
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
				return hex, i, err
			}
			hex = append(hex, byte(b))
			i += 2
			continue
		}

		if isHexDigit(slice[i]) && slice[i+1] == '>' {
			b, err := strconv.ParseUint(string(slice[i])+"0", 16, 8)
			if err != nil {
				return hex, i, err
			}
			hex = append(hex, byte(b))
			i += 2
			break
		}
	}

	return hex, i, nil
}

func parseArray(slice []byte) (Object, int, error) {
	array := make(Array, 0)

	if slice[0] != '[' {
		return array, 0, errors.New("not an array")
	}

	i := 1
	for i < len(slice) {
		if isWhitespace(slice[i]) {
			i++
			continue
		}

		if slice[i] == ']' {
			return array, i + 1, nil
		}

		object, n, err := parseObject(slice[i:])
		if err != nil {
			return array, i, err
		}
		i += n

		array = append(array, object)
	}

	return array, i, errors.New("end of array not found")
}

func parseNull(slice []byte) (Object, int, error) {
	n, ok := match(slice, "null")
	if ok {
		return Null{}, n, nil
	}

	return Null{}, 0, errors.New("not a Null")
}

func parseObjectReference(slice []byte) (Object, int, error) {
	objref := ObjectReference{}
	i := 0

	objectNumber, n, err := parseNumeric(slice[i:])
	i += n
	if err != nil {
		return objref, i, err
	}
	integer, ok := objectNumber.(Integer)
	if !ok {
		return objref, i, errors.New("expected object number not an integer")
	}
	objref.ObjectNumber = uint(integer)

	generationNumber, n, err := parseNumeric(slice[i:])
	i += n
	if err != nil {
		return objref, i, err
	}
	integer, ok = generationNumber.(Integer)
	if !ok {
		return objref, i, errors.New("expected generation number not an integer")
	}
	objref.GenerationNumber = uint(integer)

	n, ok = match(slice[i:], "R")
	i += n
	if !ok {
		return objref, i, errors.New("could not find end of object reference")
	}

	return objref, i, nil
}

func parseIndirectObject(slice []byte) (Object, int, error) {
	var io IndirectObject
	i := 0
	var err error

	// Object Number
	token, n := nextToken(slice[i:])
	objectNumber, err := strconv.ParseUint(string(token), 10, 64)
	i += n
	if err != nil {
		return io, i, err
	}
	io.ObjectNumber = uint(objectNumber)

	// Generation Number
	token, n = nextToken(slice[i:])
	generationNumber, err := strconv.ParseUint(string(token), 10, 64)
	i += n
	if err != nil {
		return io, i, err
	}
	io.GenerationNumber = uint(generationNumber)

	// "obj"
	n, ok := match(slice[i:], "obj")
	i += n
	if !ok {
		return io, i, errors.New("could not find 'obj'")
	}

	// the object
	object, n, err := parseObject(slice[i:])
	i += n
	io.Object = object
	if err != nil {
		return io, i, err
	}

	// "endobj"
	n, ok = match(slice[i:], "endobj")
	i += n
	if !ok {
		return io, i, errors.New("could not find 'endobj'")
	}

	return io, i, nil
}
