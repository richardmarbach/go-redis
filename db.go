package redis

import (
	"sync"
	"time"
)

type entryType uint8

const (
	stringEntry entryType = iota
	intEntry
)

type entry struct {
	t      entryType
	expiry int64
	value  []byte
}

// DB is the core in-memory database
type DB struct {
	entries map[string]string
	expiry  map[string]int64
	mu      sync.RWMutex
}

// NewDB creates a new in memory database
func NewDB() *DB {
	return &DB{
		entries: make(map[string]string, 1_000_000),
		expiry:  make(map[string]int64, 1_000),
	}
}

// Set a key in the database
func (db *DB) Set(key, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.entries[key] = value
}

// SetWithExpiry sets a key with expiration time in milliseconds.
// Expired values are removed the next time they're accessed.
func (db *DB) SetWithExpiry(key, value string, expiry int64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.entries[key] = value
	db.expiry[key] = (now() + expiry)
}

// Get a value for the given key
func (db *DB) Get(key string) (string, bool) {
	db.mu.RLock()
	if expiry, found := db.expiry[key]; found {
		if expiry < now() {
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

func now() int64 {
	return time.Now().UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
