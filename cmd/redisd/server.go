package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

const DefaultAddr = ":6379"

type RedisServer struct {
	db *DB
	ln net.Listener
}

func (s *RedisServer) Addr() string {
	return s.ln.Addr().String()
}

func (s *RedisServer) Listen(addr string) (err error) {
	if addr == "" {
		addr = DefaultAddr
	}

	s.ln, err = net.Listen("tcp", addr)
	log.Printf("listening on %s\n", addr)
	return err
}

func (s *RedisServer) Serve() (err error) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return nil
		}
		go s.handleConnection(conn)
	}
}

func (s *RedisServer) ListenAndServe(addr string) (err error) {
	err = s.Listen(addr)
	if err != nil {
		return err
	}
	return s.Serve()
}

func (s *RedisServer) Close() (err error) {
	return s.ln.Close()
}

func (s *RedisServer) handleConnection(conn net.Conn) {
	client := &client{NewRESPReader(conn), NewRESPWriter(conn), s.db}

	for {
		err := client.runCommand(conn)
		if err != nil {
			// Client closed the connection
			if err == io.EOF {
				return
			}
			// If we can' write errors then the connection has been closed
			if err := client.w.WriteError(err); err != nil {
				return
			}
			client.Clear()
		}
	}
}

type client struct {
	r  *RESPReader
	w  *RESPWriter
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
