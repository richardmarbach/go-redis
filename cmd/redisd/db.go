package main

import (
	"sync"
	"time"
)

func Now() int64 {
	return time.Now().UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

type DB struct {
	entries map[string]string
	expiry  map[string]int64
	mu      sync.RWMutex
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

func (db *DB) Get(key string) (string, bool) {
	db.mu.RLock()
	if expiry, found := db.expiry[key]; found {
		if expiry < Now() {
			db.mu.RUnlock()
			db.mu.Lock()
			delete(db.expiry, key)
			delete(db.entries, key)
			db.mu.Unlock()
			return "", false
		}
	}
	defer db.mu.RUnlock()

	if value, found := db.entries[key]; found {
		return value, found
	}
	return "", false
}
