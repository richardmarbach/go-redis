package resp

import "errors"

const (
	ErrorPrefix        = '-'
	ArrayPrefix        = '*'
	SimpleStringPrefix = '+'
	BulkStringPrefix   = '$'

	// Smalled valid object is ":0\r\n"
	minObjectLen = 4
)

var (
	ErrSyntax = errors.New("resp: syntax error")

	// Common responses
	PONG = NewSimpleString("PONG")

	lineSuffix = []byte("\r\n")
)
