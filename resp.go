package redis

import "errors"

const (
	RESPArray      = '*'
	RESPBulkString = '$'
)

var (
	ErrInvalidSyntax      = errors.New("resp: invalid syntax")
	ErrUnsupportedCommand = errors.New("resp: unsupported command")
)
