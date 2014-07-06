package pdf

import (
	"testing"
)

func TestBytesToInt(t *testing.T) {
	type test struct {
		b []byte
		v int
	}
	tests := []test{
		test{[]byte{0x0, 0x0, 0x0}, 0},
		test{[]byte{0x0, 0x0, 0x01}, 1},
		test{[]byte{0x0, 0x01, 0x0}, 256},
		test{[]byte{0x01, 0x0, 0x0}, 65536},
		test{[]byte{0x01, 0x01, 0x0}, 65792},
	}

	for i, test := range tests {
		result := bytesToInt(test.b)
		if result != test.v {
			t.Errorf("%d: expected %v for %#v, got %v", i, test.v, test.b, result)
		}
	}
}
