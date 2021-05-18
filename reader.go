package redis

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type RESPReader struct {
	r *bufio.Reader
}

func NewRESPReader(r io.Reader) *RESPReader {
	return &RESPReader{bufio.NewReader(r)}
}

func (r *RESPReader) Clear() {
	r.r.Discard(r.r.Buffered())
}

func (r *RESPReader) ReadType() (byte, error) {
	b, err := r.r.ReadByte()
	if err != nil {
		return b, err
	}
	switch b {
	case RESPArray, RESPBulkString:
		return b, nil
	}
	return b, ErrInvalidSyntax
}

func (r *RESPReader) ReadCommand() (string, int, error) {
	t, err := r.ReadType()
	if err != nil || t != RESPArray {
		return "", 0, ErrInvalidSyntax
	}

	n, err := r.ReadSize()
	if err != nil || n < 1 {
		return "", 0, ErrInvalidSyntax
	}

	cmd, err := r.ReadBulkString()
	if err != nil {
		return "", 0, err
	}

	return strings.ToLower(cmd), n - 1, nil
}

func (r *RESPReader) ReadSize() (int, error) {
	line, err := r.r.ReadBytes('\n')
	if err != nil {
		return 0, err
	}

	if len(line) == 4 && line[0] == '-' && line[1] == '1' && line[2] == '\r' {
		return -1, nil
	}

	var n int

	for i := range line {
		if line[i] < '0' || line[i] > '9' {
			if line[i] == '\r' && i+2 == len(line) {
				return n, nil
			}
			return 0, ErrInvalidSyntax
		}

		n = (n * 10) + int(line[i]-'0')
	}
	return 0, ErrInvalidSyntax
}

func (r *RESPReader) ReadInt64() (int64, error) {
	s, err := r.ReadBulkString()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ErrInvalidSyntax
	}
	return n, nil
}

func (r *RESPReader) ReadBulkString() (string, error) {
	t, err := r.ReadType()
	if err != nil {
		return "", err
	} else if t != RESPBulkString {
		return "", ErrInvalidSyntax
	}

	n, err := r.ReadSize()
	if err != nil {
		return "", err
	} else if n < 1 {
		return "", ErrInvalidSyntax
	}

	str := make([]byte, n+2)
	_, err = io.ReadFull(r.r, str)
	if err != nil {
		return "", err
	} else if str[len(str)-2] != '\r' || str[len(str)-1] != '\n' {
		return "", ErrInvalidSyntax
	}

	return string(str[:len(str)-2]), nil
}
