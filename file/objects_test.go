package file

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// returns nil when equivalent
func compare(got, expected interface{}) error {
	if !reflect.DeepEqual(got, expected) {
		return errors.New(fmt.Sprintf("\nExpected:\n\t%v\nGot:\n\t%v", expected, got))
	}
	return nil
}

// §7.3.10 Example 1
func TestIndirectObjectsExample1(t *testing.T) {
	io, err := ParseIndirectObject([]byte("12 0 obj\n\t(Brillig)\nendobj"))
	if err != nil {
		t.Error(err)
	}

	err = compare(io, &IndirectObject{
		ObjectNumber:     12,
		GenerationNumber: 0,
		Object:           String("Brillig"),
	})
	if err != nil {
		t.Error("Example1", err)
	}
}

// §7.3.4.2 Example 1
func TestLiteralStringExample1(t *testing.T) {
	strings := [][]byte{
		[]byte("(This is a string)"),
		[]byte("(Strings may contain newlines\nand such.)"),
		[]byte("(Strings may contain balanced parentheses () and\nspecial characters (*!&}^% and so on).)"),
		[]byte("(The following is an empty string.)"),
		[]byte("()"),
		[]byte("(It has zero (0) length.)"),
	}

	for n, slice := range strings {
		str, err := ParseLiteralString(slice)
		if err != nil {
			t.Error(n, err)
		}

		// should work because the test cases are encoded as Go strings...
		err = compare(str, String(slice[1:len(slice)-1]))
		if err != nil {
			t.Error(n, err)
		}
	}

}
