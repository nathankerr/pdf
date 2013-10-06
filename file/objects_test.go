package file

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"testing"
)

type test struct {
	literal []byte
	object  Object
}

// general test runner for parse function tests
func runTests(t *testing.T, tests []test) {
	pc, _, line, ok := runtime.Caller(1)
	caller := "UNABLE TO DETERMINE CALLER"
	if ok {
		fn := runtime.FuncForPC(pc)
		functionName := path.Ext(path.Base(fn.Name()))[1:]
		caller = fmt.Sprintf("%v (line %v)", functionName, line)
	}

	for n, test := range tests {
		var fn parseFn
		switch test.object.(type) {
		case IndirectObject:
			fn = ParseIndirectObject
		default:
			fn = ParseObject
		}

		object, length, err := fn(test.literal)
		if err != nil {
			t.Errorf("%v test %v\nParse Error:\n\t%v\n", caller, n, err)
		}

		if length != len(test.literal) {
			t.Errorf("%v test %v\nExpected Length:\n\t%v\nGot Length:\n\t%v\n", caller, n, len(test.literal), length)
		}

		if !reflect.DeepEqual(object, test.object) {
			t.Errorf("%v test %v\nExpected Object:\n\t%#v\nGot Object:\n\t%#v\n", caller, n, test.object, object)
		}
	}
}

// returns nil when equivalent
func compare(got, expected interface{}) error {
	if !reflect.DeepEqual(got, expected) {
		return errors.New(fmt.Sprintf("\nExpected:\n\t%#v\nGot:\n\t%#v", expected, got))
	}
	return nil
}

// §7.3.10 Example 1
func TestIndirectObjectsExample1(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("12 0 obj\n\t(Brillig)\nendobj"),
			object: IndirectObject{
				ObjectNumber:     12,
				GenerationNumber: 0,
				Object:           Object(String("Brillig")),
			},
		},
	})
}

// §7.3.4.2 Example 1
func TestLiteralStringExample1(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("(This is a string)"),
			object:  String("This is a string"),
		},
		test{
			literal: []byte("(Strings may contain newlines\nand such.)"),
			object:  String("Strings may contain newlines\nand such."),
		},
		test{
			literal: []byte("(Strings may contain balanced parentheses () and\nspecial characters (*!&}^% and so on).)"),
			object:  String("Strings may contain balanced parentheses () and\nspecial characters (*!&}^% and so on)."),
		},
		test{
			literal: []byte("(The following is an empty string.)"),
			object:  String("The following is an empty string."),
		},
		test{
			literal: []byte("()"),
			object:  String(""),
		},
		test{
			literal: []byte("(It has zero (0) length.)"),
			object:  String("It has zero (0) length."),
		},
	})
}

// §7.3.4.2 Example 2
func TestLiteralStringExample2(t *testing.T) {
	first, _, err := ParseLiteralString([]byte("(These \\\ntwo strings \\\nare the same.)"))
	if err != nil {
		t.Error(err)
	}
	second, _, err := ParseLiteralString([]byte("(These two strings are the same.)"))
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
	runTests(t, []test{
		// Example 3
		test{
			literal: []byte("(This string has an end-of-line at the end of it.\n)"),
			object:  String("This string has an end-of-line at the end of it.\n"),
		},
		test{
			literal: []byte("(So does this one.\n)"),
			object:  String("So does this one.\n"),
		},
		// Example 4
		test{
			literal: []byte("(This string contains \\245two octal characters\\307.)"),
			object:  String("This string contains \\245two octal characters\\307."),
		},
		// Example 5
		test{
			literal: []byte("(\\0053)"),
			object:  String("\\0053"),
		},
		test{
			literal: []byte("(\\053)"),
			object:  String("\\053"),
		},
		test{
			literal: []byte("(\\53)"),
			object:  String("\\53"),
		},
	})
}

//§7.3.7 Example
func TestDictionaryExample(t *testing.T) {
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
		Name("Type"):        Name("Example"),
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
	runTests(t, []test{
		test{
			literal: []byte("/Name1"),
			object:  Name("Name1"),
		},
		test{
			literal: []byte("/ASomewhatLongerName"),
			object:  Name("ASomewhatLongerName"),
		},
		test{
			literal: []byte("/A;Name_With-Various***Characters?"),
			object:  Name("A;Name_With-Various***Characters?"),
		},
		test{
			literal: []byte("/1.2"),
			object:  Name("1.2"),
		},
		test{
			literal: []byte("/$$"),
			object:  Name("$$"),
		},
		test{
			literal: []byte("/@pattern"),
			object:  Name("@pattern"),
		},
		test{
			literal: []byte("/.notdef"),
			object:  Name(".notdef"),
		},
		test{
			// this example as defined would not work as
			// the capitalization of "lime" was different
			// in the literal and the result
			// Fixed by making consistent
			literal: []byte("/Lime#20Green"),
			object:  Name("Lime Green"),
		},
		test{
			literal: []byte("/paired#28#29parentheses"),
			object:  Name("paired()parentheses"),
		},
		test{
			literal: []byte("/The_Key_of_F#23_Minor"),
			object:  Name("The_Key_of_F#_Minor"),
		},
		test{
			literal: []byte("/A#42"),
			object:  Name("AB"),
		},
	})
}

// §7.3.2
func TestBoolean(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("true"),
			object:  Boolean(true),
		},
		test{
			literal: []byte("false"),
			object:  Boolean(false),
		},
	})
}

//§7.3.3
func TestNumericObjects(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("123"),
			object:  Integer(123),
		},
		test{
			literal: []byte("43445"),
			object:  Integer(43445),
		},
		test{
			literal: []byte("+17"),
			object:  Integer(17),
		},
		test{
			literal: []byte("-98"),
			object:  Integer(-98),
		},
		test{
			literal: []byte("0"),
			object:  Integer(0),
		},
		test{
			literal: []byte("34.5"),
			object:  Real(34.5),
		},
		test{
			literal: []byte("-3.62"),
			object:  Real(-3.62),
		},
		test{
			literal: []byte("+123.6"),
			object:  Real(123.6),
		},
		test{
			literal: []byte("4."),
			object:  Real(4),
		},
		test{
			literal: []byte("-.002"),
			object:  Real(-.002),
		},
		test{
			literal: []byte("0.0"),
			object:  Real(0.0),
		},
	})
}

//§7.3.4.3 Examples 1, 2
func TestHexadecimalStringExamples12(t *testing.T) {
	runTests(t, []test{
		// Example 1
		test{
			literal: []byte("<4E6F762073686D6F7A206B6120706F702E>"),
			object:  String{0x4E, 0x6F, 0x76, 0x20, 0x73, 0x68, 0x6D, 0x6F, 0x7A, 0x20, 0x6B, 0x61, 0x20, 0x70, 0x6F, 0x70, 0x2E},
		},
		// Example 2
		test{
			literal: []byte("<901FA3>"),
			object:  String{0x90, 0x1F, 0xA3},
		},
		test{
			literal: []byte("<901FA>"),
			object:  String{0x90, 0x1F, 0xA0},
		},
	})
}

//§7.3.6
func TestArrayExample(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("[549 3.14 false (Ralph) /SomeName]"),
			object: Array{
				Integer(549),
				Real(3.14),
				Boolean(false),
				String("Ralph"),
				Name("SomeName"),
			},
		},
	})
}

//§7.3.9
func TestNull(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("null"),
			object:  Null{},
		},
	})
}
