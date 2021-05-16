package resp

import "testing"

type respLenTest struct {
	given         []byte
	length        int
	endIndex      int
	errorExpected bool
}

func TestParseLineLen(t *testing.T) {
	cases := []respLenTest{
		// Invalid lines
		{[]byte{}, 0, -1, true},
		{[]byte(""), 0, -1, true},
		{[]byte("-\r\n"), 0, -1, true},
		{[]byte("-OK\r\n"), 0, -1, true},
		{[]byte("$0x2\r\n"), 0, -1, true},
		{[]byte("$-19\r\n"), 0, -1, true},
		{[]byte("$1"), 0, -1, true},
		{[]byte("$1\r"), 0, -1, true},

		// Valid lines
		{[]byte("$-1\r\n"), -1, 4, false},
		{[]byte("$1\r\n"), 1, 3, false},
		{[]byte("$100\r\n"), 100, 5, false},
		{[]byte("$9876\r\n"), 9876, 6, false},
		{[]byte("$1\r\n$100\r\n"), 1, 3, false},
	}

	for i, tc := range cases {
		length, endIndex, err := parseLineLen(tc.given)

		if tc.errorExpected {
			if err == nil {
				t.Errorf("[%d] expected an error buf didn't get one", i)
			}
		} else {
			if length != tc.length {
				t.Errorf("[%d] expected length %v, got %v", i, tc.length, length)
			}

			if endIndex != tc.endIndex {
				t.Errorf("[%d] expected endIndex %v, got %v", i, tc.endIndex, endIndex)
			}
		}

	}
}
