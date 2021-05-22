package redis

import (
	"context"
	"encoding/binary"
	"sync"
	"time"
)

const (
	// NoExpire is a sentinel value indicating that the entry shouldn't expire
	NoExpire = -1
)

type entryType uint8

const (
	stringEntry entryType = iota
	intEntry
)

type dbEntry struct {
	t      entryType
	expiry int64
	value  []byte
}

func NewStringEntry(value string, expiry int64) dbEntry {
	return dbEntry{
		t:      stringEntry,
		expiry: expiry,
		value:  []byte(value),
	}
}

func NewInt64Entry(value int64, expiry int64) dbEntry {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(value))

	return dbEntry{
		t:      intEntry,
		expiry: expiry,
		value:  buf,
	}
}

func (e *dbEntry) Type() entryType {
	return e.t
}

func (e *dbEntry) SetString(s string) {
	e.value = []byte(s)
}

func (e *dbEntry) String() string {
	return string(e.value)
}

func (e *dbEntry) Int64() int64 {
	return int64(binary.LittleEndian.Uint64(e.value))
}

func (e *dbEntry) SetInt64(value int64) {
	binary.LittleEndian.PutUint64(e.value, uint64(value))
}

// DB is the core in-memory database
type DB struct {
	entries map[string]dbEntry
	mu      URWMutex

	// Background task management
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewDB creates a new in memory database
func NewDB() *DB {
	db := &DB{
		entries:  make(map[string]dbEntry, 1_000),
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
	db.entries[key] = NewStringEntry(value, NoExpire)
	db.mu.Unlock()
}

// SetWithExpiry sets a key with expiration time in milliseconds.
// Expired values are removed the next time they're accessed.
func (db *DB) SetWithExpiry(key, value string, expiry int64) {
	db.mu.Lock()
	db.entries[key] = NewStringEntry(value, (now() + expiry))
	db.mu.Unlock()
}

// Get a value for the given key
func (db *DB) Get(key string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	entry, found := db.entries[key]

	if !found {
		return "", false
	}

	if entry.expiry > 0 && entry.expiry < now() {
		db.mu.Upgrade()
		delete(db.entries, key)
		db.mu.Downgrade()
		return "", false
	}

	return entry.String(), found
}

func (db *DB) Incr(key string, amount int64) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	entry, found := db.entries[key]
	if !found {
		entry = NewInt64Entry(0, NoExpire)
	}

	entry.SetInt64(entry.Int64() + amount)

	return entry.Int64(), nil
}

func now() int64 {
	return time.Now().UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
