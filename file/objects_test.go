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
				Object:           String("Brillig"),
			},
		},
		test{
			literal: []byte("\t(Brillig)"),
			object:  String("Brillig"),
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
	runTests(t, []test{
		test{
			literal: []byte(`<< /Type /Example
/Subtype /DictionaryExample
/Version 0.01
/Integeritem 12
/StringItem (a string)
/Subdictionary << /Item1 0.4
				  /Item2 true
				  /LastItem (not!)
				  /VeryLastItem (OK)
			   >>
>>`),
			object: Dictionary{
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
			},
		},
		// again, but without unneeded spaces
		test{
			literal: []byte(`<</Type/Example/Subtype/DictionaryExample/Version 0.01/Integeritem 12/StringItem (a string)/Subdictionary << /Item1 0.4/Item2 true/LastItem (not!)/VeryLastItem (OK)>>>>`),
			object: Dictionary{
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
			},
		},
	})
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

// Cross reference stream from spec without stream
func TestSpecificationsCrossRefStream(t *testing.T) {
	runTests(t, []test{
		test{
			literal: []byte("124348 0 obj\r<</DecodeParms<</Columns 5/Predictor 12>>/Filter/FlateDecode/ID[<9597C618BC90AFA4A078CA72B2DD061C><48726007F483D547A8BEFF6E9CDA072F>]/Index[124332 848]/Info 124331 0 R/Length 137/Prev 8983958/Root 124333 0 R/Size 125180/Type/XRef/W[1 3 1]>>stream\r\nh\xde\xecұ\r\x82`\x10\x86\xe1\xbb?t\x82Rа\x00\xee\x00\v\xb8\x83\v\xb8\x8a\xb5\xb5\x03\x98\xb8\b\x83P\x90XX\x98X`\x82|7\x80\xc6ּ͓+.W\\\xde\xe4f\xa5%\xb3\xedM\xfa#|\xcal/\xd3A\xae\xaer3\xc8\xf5I\x16Ux\f\xeb\xd0e\xde\xc6\xfc\x92U/wwK\xeeC\xa7y\xb9\xfd\xc9l\xfa\xbe\x83\xf8\xab~\xe6\x0fHWHWHW\x88t\x85t\x85t\x85\xf8\x87]\xcdss\x19\xdf\x02\f\x00\x8d=\x1f\x11\r\nendstream\rendobj"),
			object: IndirectObject{
				ObjectNumber:     124348,
				GenerationNumber: 0,
				Object: Stream{
					Dictionary: Dictionary{
						Name("DecodeParms"): Dictionary{
							Name("Columns"):   Integer(5),
							Name("Predictor"): Integer(12),
						},
						Name("Filter"): Name("FlateDecode"),
						Name("ID"): Array{
							String([]byte{0x95, 0x97, 0xC6, 0x18, 0xBC, 0x90, 0xAF, 0xA4, 0xA0, 0x78, 0xCA, 0x72, 0xB2, 0xDD, 0x06, 0x1C}),
							String([]byte{0x48, 0x72, 0x60, 0x07, 0xF4, 0x83, 0xD5, 0x47, 0xA8, 0xBE, 0xFF, 0x6E, 0x9C, 0xDA, 0x07, 0x2F}),
						},
						Name("Index"): Array{
							Integer(124332),
							Integer(848),
						},
						Name("Info"): ObjectReference{
							ObjectNumber:     124331,
							GenerationNumber: 0,
						},
						Name("Length"): Integer(137),
						Name("Prev"):   Integer(8983958),
						Name("Root"): ObjectReference{
							ObjectNumber:     124333,
							GenerationNumber: 0,
						},
						Name("Size"): Integer(125180),
						Name("Type"): Name("XRef"),
						Name("W"): Array{
							Integer(1),
							Integer(3),
							Integer(1),
						},
					},
					Stream: []byte{0x68, 0xde, 0xec, 0xd2, 0xb1, 0xd, 0x82, 0x60, 0x10, 0x86, 0xe1, 0xbb, 0x3f, 0x74, 0x82, 0x52, 0xd0, 0xb0, 0x0, 0xee, 0x0, 0xb, 0xb8, 0x83, 0xb, 0xb8, 0x8a, 0xb5, 0xb5, 0x3, 0x98, 0xb8, 0x8, 0x83, 0x50, 0x90, 0x58, 0x58, 0x98, 0x58, 0x60, 0x82, 0x7c, 0x37, 0x80, 0xc6, 0xd6, 0xbc, 0xcd, 0x93, 0x2b, 0x2e, 0x57, 0x5c, 0xde, 0xe4, 0x66, 0xa5, 0x25, 0xb3, 0xed, 0x4d, 0xfa, 0x23, 0x7c, 0xca, 0x6c, 0x2f, 0xd3, 0x41, 0xae, 0xae, 0x72, 0x33, 0xc8, 0xf5, 0x49, 0x16, 0x55, 0x78, 0xc, 0xeb, 0xd0, 0x65, 0xde, 0xc6, 0xfc, 0x92, 0x55, 0x2f, 0x77, 0x77, 0x4b, 0xee, 0x43, 0xa7, 0x79, 0xb9, 0xfd, 0xc9, 0x6c, 0xfa, 0xbe, 0x83, 0xf8, 0xab, 0x7e, 0xe6, 0xf, 0x48, 0x57, 0x48, 0x57, 0x48, 0x57, 0x88, 0x74, 0x85, 0x74, 0x85, 0x74, 0x85, 0xf8, 0x87, 0x5d, 0xcd, 0x73, 0x73, 0x19, 0xdf, 0x2, 0xc, 0x0, 0x8d, 0x3d, 0x1f, 0x11},
				},
			},
		},
	})
}
