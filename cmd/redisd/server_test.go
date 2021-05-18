package main

import (
	"net"
	"testing"
)

func TestRedisServer(t *testing.T) {

	t.Run("PING", func(t *testing.T) {
		client := NewServer(t)

		_, err := client.Write([]byte("*1\r\n$4\r\nping\r\n"))
		if err != nil {
			t.Fatal(err)
		}

		response := make([]byte, 40)
		_, err = client.Read(response)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("PING with arg", func(t *testing.T) {
		client := NewServer(t)

		_, err := client.Write([]byte("*2\r\n$4\r\nping\r\n\"abc\"\r\n"))
		if err != nil {
			t.Fatal(err)
		}

		response := make([]byte, 40)
		_, err = client.Read(response)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func NewServer(t testing.TB) net.Conn {
	s := RedisServer{}
	err := s.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer s.Close()
		// Use next available port
		if err := s.Serve(); err != nil {
			t.Fatal(err)
		}
	}()

	client, err := net.Dial("tcp", s.Addr())
	if err != nil {
		t.Fatal(err)
	}

	return client
}
