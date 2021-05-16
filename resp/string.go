package resp

import (
	"bytes"
	"strconv"
)

type String []byte

func NewSimpleString(s string) String {
	var buf bytes.Buffer
	buf.WriteByte(SimpleStringPrefix)
	buf.WriteString(s)
	buf.Write(lineSuffix)
	return String(buf.Bytes())
}

func NewBulkString(s string) String {
	var buf bytes.Buffer
	buf.WriteByte(BulkStringPrefix)
	buf.WriteString(strconv.Itoa(len(s)))
	buf.Write(lineSuffix)
	buf.WriteString(s)
	buf.Write(lineSuffix)
	return String(buf.Bytes())
}

func (s String) Raw() []byte {
	return s
}

func (s String) Slice() []byte {
	if s[0] == SimpleStringPrefix {
		return s[1 : len(s)-2]
	}

	length, lengthIndexEnd, err := parseLineLen(s)
	if err != nil || length == -1 {
		return nil
	}
	return s[lengthIndexEnd+1 : len(s)-2]
}

func (s String) String() string {
	return string(s.Slice())
}
