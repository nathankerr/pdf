package pdf

import (
	"reflect"
	"testing"
)

func TestBytesToInt(t *testing.T) {
	type test struct {
		b []byte
		v uint
	}
	tests := []test{
		test{[]byte{0x0, 0x0, 0x0}, 0},
		test{[]byte{0x0, 0x0, 0x0, 0x01}, 1},
		test{[]byte{0x0, 0x0, 0x01, 0x0}, 256},
		test{[]byte{0x0, 0x01, 0x0, 0x0}, 65536},
		test{[]byte{0x0, 0x01, 0x01, 0x0}, 65792},
	}

	for i, test := range tests {
		result := bytesToInt(test.b)
		if result != test.v {
			t.Errorf("%d: expected %v for %#v, got %v", i, test.v, test.b, result)
		}
	}
}

func TestNBytesForInt(t *testing.T) {
	type test struct {
		b []byte
		v int
	}
	tests := []test{
		test{[]byte{0x0}, 0},
		test{[]byte{0x01}, 1},
		test{[]byte{0x01, 0x0}, 256},
		test{[]byte{0x01, 0x0, 0x0}, 65536},
		test{[]byte{0x01, 0x01, 0x0}, 65792},
	}

	for i, test := range tests {
		result := nBytesForInt(test.v)
		if result != len(test.b) {
			t.Errorf("%d: expected %v for %#v, got %v", i, len(test.b), test.v, result)
		}
	}
}

func TestIntToBytes(t *testing.T) {
	type test struct {
		b []byte
		v uint
	}
	tests := []test{
		test{[]byte{0x0}, 0},
		test{[]byte{0x01}, 1},
		test{[]byte{0x01, 0x0}, 256},
		test{[]byte{0x01, 0x0, 0x0}, 65536},
		test{[]byte{0x01, 0x01, 0x0}, 65792},
	}

	for i, test := range tests {
		result := intToBytes(test.v, len(test.b))
		if !reflect.DeepEqual(result, test.b) {
			t.Errorf("%d: expected %v for %#v, got %v", i, test.b, test.v, result)
		}
	}
}

func TestIntToBytesAndBack(t *testing.T) {
	type test struct {
		b []byte
		v uint
	}
	tests := []test{
		test{[]byte{0x0}, 0},
		test{[]byte{0x01}, 1},
		test{[]byte{0x01, 0x0}, 256},
		test{[]byte{0x01, 0x0, 0x0}, 65536},
		test{[]byte{0x01, 0x01, 0x0}, 65792},
	}

	for i, test := range tests {
		asBytes := intToBytes(test.v, 8)
		asInt := bytesToInt(asBytes)

		if asInt != test.v {
			t.Errorf("%d: expected %v for %#v, got %v", i, test.v, test.b, asInt)
		}
	}
}
