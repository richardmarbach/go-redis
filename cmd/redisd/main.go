package main

import (
	"log"

	"github.com/richardmarbach/go-redis"
)

func main() {
	server := &redis.RedisServer{
		DB: redis.NewDB(),
	}

	if err := server.ListenAndServe(redis.DefaultAddr); err != nil {
		log.Fatal(err)
	}
}
