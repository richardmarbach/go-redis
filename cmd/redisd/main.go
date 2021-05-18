package main

import (
	"log"
)

func main() {
	server := &RedisServer{
		db: NewDB(),
	}

	if err := server.ListenAndServe(DefaultAddr); err != nil {
		log.Fatal(err)
	}
}
