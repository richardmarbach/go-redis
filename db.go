package redis

import (
	"context"
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

func (e entry) String() string {
	return string(e.value)
}

// DB is the core in-memory database
type DB struct {
	entries map[string]entry
	mu      sync.RWMutex

	// Background task management
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewDB creates a new in memory database
func NewDB() *DB {
	db := &DB{
		entries:  make(map[string]entry, 1_000),
		shutdown: make(chan struct{}),
	}

	go db.startVacuum()

	return db
}

func (db *DB) startVacuum() {
	db.wg.Add(1)
	for {
		select {
		case <-db.shutdown:
			db.wg.Done()
			return
		case <-time.After(30 * time.Second):
			db.Vacuum()
		}
	}
}

// Vacuum removes expired entries
func (db *DB) Vacuum() {
	startedAt := now()
	for key, value := range db.entries {
		if value.expiry < startedAt {
			db.mu.Lock()
			delete(db.entries, key)
			db.mu.Unlock()
		}
	}
}

// Shutdown the database
func (db *DB) Shutdown(ctx context.Context) error {
	db.shutdown <- struct{}{}

	c := make(chan struct{})

	go func() {
		defer close(c)
		db.wg.Wait()
	}()

	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Set a key in the database
func (db *DB) Set(key, value string) {
	db.mu.Lock()
	db.entries[key] = entry{stringEntry, -1, []byte(value)}
	db.mu.Unlock()
}

// SetWithExpiry sets a key with expiration time in milliseconds.
// Expired values are removed the next time they're accessed.
func (db *DB) SetWithExpiry(key, value string, expiry int64) {
	db.mu.Lock()
	db.entries[key] = entry{stringEntry, (now() + expiry), []byte(value)}
	db.mu.Unlock()
}

// Get a value for the given key
func (db *DB) Get(key string) (string, bool) {
	db.mu.RLock()
	entry, found := db.entries[key]
	db.mu.RUnlock()

	if !found {
		return "", false
	}

	if entry.expiry > 0 && entry.expiry < now() {
		return "", false
	}

	return entry.String(), found
}

func now() int64 {
	return time.Now().UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
