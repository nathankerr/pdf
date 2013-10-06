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

// §7.3.4.2 Example 2
func TestLiteralStringExample2(t *testing.T) {
	first, err := ParseLiteralString([]byte("(These \\\ntwo strings \\\nare the same.)"))
	if err != nil {
		t.Error(err)
	}

	second, err := ParseLiteralString([]byte("(These two strings are the same.)"))
	if err != nil {
		t.Error(err)
	}

	err = compare(first, second)
	if err != nil {
		t.Error(err)
	}
}

// §7.3.4.2 Examples 3, 4, 5
// These examples deal with how pdf strings should
// be interpreted. There is nothing in the spec that
// says that an string with escaped characters which
// is then interpreted is equivalent to one without
// escaping those characters (with the exeption in
// Example 2). These examples are included for
// completeness.
func TestLiteralStringExamples345(t *testing.T) {
	strings := [][]byte{
		// Example 3
		[]byte("(This string has an end-of-line at the end of it.\n)"),
		[]byte("(So does this one.\n)"),
		// Example 4
		[]byte("(This string contains \\245two octal characters\\307.)"),
		// Example 5
		[]byte("(\\0053)"),
		[]byte("(\\053)"),
		[]byte("(\\53)"),
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

//§7.3.7 Example
func DISABLEDTestDictionaryExample(t *testing.T) {
	literal := []byte(`<< /Type /Example
/Subtype /DictionaryExample
/Version 0.01
/Integeritem 12
/StringItem (a string)
/Subdictionary << /Item1 0.4
				  /Item2 true
				  /LastItem (not!)
				  /VeryLastItem (OK)
			   >>
>>`)

	dict, length, err := ParseDictionary(literal)
	if err != nil {
		t.Error(err)
	}

	if length != len(literal) {
		t.Error("expected ", len(literal), ", got ", length)
	}

	err = compare(dict, Dictionary{
		Name("Type"):        Name("/Example"),
		Name("Subtype"):     Name("DictionaryExample"),
		Name("Version"):     Real(0.01),
		Name("Integeritem"): Integer(12),
		Name("StringItem"):  String("a string"),
		Name("Subdictionary"): Dictionary{
			Name("Item1"):        Real(0.4),
			Name("Item2"):        Boolean(true),
			Name("LastItem"):     String("not!"),
			Name("VeryLastItem"): String("OK"),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

//§7.3.5 Table 4
func TestNameExamples(t *testing.T) {
	type test struct {
		literal []byte
		name    Name
	}

	tests := []test{
		test{
			literal: []byte("/Name1"),
			name:    Name("Name1"),
		},
		test{
			literal: []byte("/ASomewhatLongerName"),
			name:    Name("ASomewhatLongerName"),
		},
		test{
			literal: []byte("/A;Name_With-Various***Characters?"),
			name:    Name("A;Name_With-Various***Characters?"),
		},
		test{
			literal: []byte("/1.2"),
			name:    Name("1.2"),
		},
		test{
			literal: []byte("/$$"),
			name:    Name("$$"),
		},
		test{
			literal: []byte("/@pattern"),
			name:    Name("@pattern"),
		},
		test{
			literal: []byte("/.notdef"),
			name:    Name(".notdef"),
		},
		test{
			// this example as defined would not work as
			// the capitalization of "lime" was different
			// in the literal and the result
			// Fixed by making consistent
			literal: []byte("/Lime#20Green"),
			name:    Name("Lime Green"),
		},
		test{
			literal: []byte("/paired#28#29parentheses"),
			name:    Name("paired()parentheses"),
		},
		test{
			literal: []byte("/The_Key_of_F#23_Minor"),
			name:    Name("The_Key_of_F#_Minor"),
		},
		test{
			literal: []byte("/A#42"),
			name:    Name("AB"),
		},
	}

	for n, test := range tests {
		name, length, err := ParseName(test.literal)
		if err != nil {
			t.Error(n, err)
		}

		if length != len(test.literal) {
			t.Error("expected length of", len(test.literal), "got", length)
		}

		err = compare(name, test.name)
		if err != nil {
			t.Error(n, err)
		}
	}
}

// §7.3.2
func TestBoolean(t *testing.T) {
	type test struct {
		literal []byte
		boolean Boolean
	}

	tests := []test{
		test{
			literal: []byte("true"),
			boolean: Boolean(true),
		},
		test{
			literal: []byte("false"),
			boolean: Boolean(false),
		},
	}

	for n, test := range tests {
		boolean, length, err := ParseBoolean(test.literal)
		if err != nil {
			t.Error(n, err)
		}

		if length != len(test.literal) {
			t.Error(n, "expected", len(test.literal), "got", length)
		}

		err = compare(boolean, test.boolean)
		if err != nil {
			t.Error(n, err)
		}
	}
}

//§7.3.3
func TestNumericObjects(t *testing.T) {
	type test struct {
		literal []byte
		numeric Object
	}

	tests := []test{
		test{
			literal: []byte("123"),
			numeric: Integer(123),
		},
		test{
			literal: []byte("43445"),
			numeric: Integer(43445),
		},
		test{
			literal: []byte("+17"),
			numeric: Integer(17),
		},
		test{
			literal: []byte("-98"),
			numeric: Integer(-98),
		},
		test{
			literal: []byte("0"),
			numeric: Integer(0),
		},
		test{
			literal: []byte("34.5"),
			numeric: Real(34.5),
		},
		test{
			literal: []byte("-3.62"),
			numeric: Real(-3.62),
		},
		test{
			literal: []byte("+123.6"),
			numeric: Real(123.6),
		},
		test{
			literal: []byte("4."),
			numeric: Real(4),
		},
		test{
			literal: []byte("-.002"),
			numeric: Real(-.002),
		},
		test{
			literal: []byte("0.0"),
			numeric: Real(0.0),
		},
	}

	for n, test := range tests {
		numeric, length, err := ParseNumeric(test.literal)
		if err != nil {
			t.Error(n, err)
		}

		if length != len(test.literal) {
			t.Error(n, "expected", len(test.literal), "got", length)
		}

		err = compare(numeric, test.numeric)
		if err != nil {
			t.Error(n, err)
		}
	}
}
