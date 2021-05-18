package redis

import (
	"bytes"
	"io"
	"net"
	"strconv"
)

type RESPWriter struct {
	io.Writer
}

func NewRESPWriter(w io.Writer) *RESPWriter {
	return &RESPWriter{
		Writer: w,
	}
}

func (w *RESPWriter) WriteError(err error) error {
	var buf bytes.Buffer
	buf.WriteByte('-')
	buf.WriteString(err.Error())
	buf.Write([]byte{'\r', '\n'})
	_, err = w.Write(buf.Bytes())
	return err
}

func writeError(conn net.Conn, err error) error {
	var buf bytes.Buffer
	buf.WriteByte('-')
	buf.WriteString(err.Error())
	buf.Write([]byte{'\r', '\n'})
	_, err = conn.Write(buf.Bytes())
	return err
}

func writeSimpleString(conn net.Conn, msg string) error {
	var buf bytes.Buffer
	buf.WriteByte('+')
	buf.WriteString(msg)
	buf.Write([]byte{'\r', '\n'})
	_, err := conn.Write(buf.Bytes())
	return err
}

type BulkString []byte

func NewBulkString(msg string) BulkString {
	var buf bytes.Buffer
	buf.WriteByte('$')
	buf.WriteString(strconv.Itoa(len(msg)))
	buf.Write([]byte{'\r', '\n'})

	buf.WriteString(msg)
	buf.Write([]byte{'\r', '\n'})

	return BulkString(buf.Bytes())
}

func writeBulkString(conn net.Conn, msg string) error {
	var buf bytes.Buffer
	buf.WriteByte('$')
	buf.WriteString(strconv.Itoa(len(msg)))
	buf.Write([]byte{'\r', '\n'})
	buf.WriteString(msg)
	buf.Write([]byte{'\r', '\n'})
	_, err := conn.Write(buf.Bytes())
	return err
}

func writeNilString(conn net.Conn) error {
	_, err := conn.Write([]byte("$-1\r\n"))
	return err
}
