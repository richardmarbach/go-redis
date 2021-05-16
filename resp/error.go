package resp

import "bytes"

type Error []byte

func NewError(msg string) Error {
	var buf bytes.Buffer
	buf.WriteByte(ErrorPrefix)
	buf.WriteString(msg)
	buf.Write(lineSuffix)
	return Error(buf.Bytes())
}

func (e Error) Raw() []byte {
	return e
}

func (e Error) Slice() []byte {
	return e[1 : len(e)-2]
}

func (e Error) Error() string {
	return string(e.Slice())
}
