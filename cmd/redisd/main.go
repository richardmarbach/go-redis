package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	server := &RedisServer{
		db: NewDB(),
	}

	if err := server.ListenAndServe(DefaultAddr); err != nil {
		log.Fatal(err)
	}
}

const DefaultAddr = ":6379"

type RedisServer struct {
	db    *DB
	ln    net.Listener
	Ready bool
}

func (s *RedisServer) Addr() string {
	return s.ln.Addr().String()
}

func (s *RedisServer) ListenAndServe(addr string) (err error) {
	if addr == "" {
		addr = DefaultAddr
	}

	s.ln, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Printf("listening on %s\n", addr)

	s.Ready = true

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return nil
		}
		go s.handleConnection(conn)
	}
}

func (s *RedisServer) Close() (err error) {
	return s.ln.Close()
}

func (s *RedisServer) handleConnection(conn net.Conn) {
	client := &client{NewRESPReader(conn), s.db}

	for {
		err := client.runCommand(conn)
		if err != nil {
			// Client closed the connection
			if err == io.EOF {
				return
			}
			// If we can' write errors then the connection has been closed
			if err := writeError(conn, err); err != nil {
				return
			}
			client.Clear()
		}
	}
}

type client struct {
	r  *RESPReader
	db *DB
}

func (c *client) Clear() {
	c.r.Clear()
}

func (c *client) runCommand(conn net.Conn) error {
	cmd, args, err := c.r.ReadCommand()
	if err != nil {
		return err
	}

	fmt.Printf("%s %d\n", cmd, args)

	switch cmd {
	case "set":
		return c.handleSet(conn, args)
	case "get":
		return c.handleGet(conn, args)
	case "ping":
		return c.handlePing(conn, args)
	case "echo":
		return c.handleEcho(conn, args)
	case "quit":
		writeSimpleString(conn, "OK")
		return conn.Close()
	default:
		return ErrUnsupportedCommand
	}
}

func (c *client) handleSet(conn net.Conn, args int) error {
	if args < 2 {
		return ErrInvalidSyntax
	}

	key, err := c.r.ReadBulkString()
	if err != nil {
		return err
	}
	value, err := c.r.ReadBulkString()
	if err != nil {
		return err
	}

	opts, err := c.readSetOpts(args - 2)
	if err != nil {
		return err
	}

	if expiry, found := opts["px"]; found {
		c.db.SetWithExpiry(key, value, expiry)
	} else {
		c.db.Set(key, value)
	}

	return writeSimpleString(conn, "OK")
}

func (c *client) readSetOpts(n int) (map[string]int64, error) {
	if n < 1 {
		return nil, nil
	}

	opts := make(map[string]int64, n)

	for i := 0; i < n; i++ {
		opt, err := c.r.ReadBulkString()
		if err != nil {
			return nil, err
		}

		opt = strings.ToLower(opt)

		switch opt {
		case "px":
			i++
			n, err := c.r.ReadInt64()
			if err != nil {
				return nil, err
			}
			opts[opt] = n
		default:
			return nil, ErrInvalidSyntax
		}
	}

	return opts, nil
}

func (c *client) handleGet(conn net.Conn, args int) error {
	if args != 1 {
		return ErrInvalidSyntax
	}

	key, err := c.r.ReadBulkString()
	if err != nil {
		return err
	}

	value, found := c.db.Get(key)
	if !found {
		return writeNilString(conn)
	}

	return writeBulkString(conn, value)
}

func (c *client) handlePing(conn net.Conn, args int) error {
	switch args {
	case 0:
		_, err := conn.Write([]byte("+PONG\r\n"))
		return err
	case 1:
		s, err := c.r.ReadBulkString()
		if err != nil {
			return err
		}
		return writeSimpleString(conn, s)
	default:
		return ErrInvalidSyntax
	}
}

func (c *client) handleEcho(conn net.Conn, args int) error {
	switch args {
	case 1:
		s, err := c.r.ReadBulkString()
		if err != nil {
			return err
		}
		return writeBulkString(conn, s)
	default:
		return ErrInvalidSyntax
	}
}

const (
	RESPArray      = '*'
	RESPBulkString = '$'
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

var ErrInvalidSyntax = errors.New("resp: invalid syntax")
var ErrUnsupportedCommand = errors.New("resp: unsupported command")

type DB struct {
	entries map[string]string
	expiry  map[string]int64
	mu      sync.Mutex
}

func NewDB() *DB {
	return &DB{
		entries: make(map[string]string, 1_000_000),
		expiry:  make(map[string]int64, 1_000),
	}
}

func (db *DB) Set(key, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.entries[key] = value
}

func (db *DB) SetWithExpiry(key, value string, expiry int64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.entries[key] = value
	db.expiry[key] = (Now() + expiry)
}

func Now() int64 {
	return time.Now().UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func (db *DB) Get(key string) (string, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if expiry, found := db.expiry[key]; found {
		if expiry < Now() {
			delete(db.expiry, key)
			delete(db.entries, key)
			return "", false
		}
	}

	if value, found := db.entries[key]; found {
		return value, found
	}
	return "", false
}
