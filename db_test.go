package redis

import (
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	db := NewDB()

	t.Run("Get returns false when the value doesn't exist", func(t *testing.T) {
		assertValueMissing(t, db, "key")
	})

	t.Run("Get returns the value when it exists", func(t *testing.T) {
		db.Set("key", "value")
		assertValueExists(t, db, "key", "value")
	})

	t.Run("Value expires after the given milliseconds", func(t *testing.T) {
		db.SetWithExpiry("key", "value", 1)
		assertValueExists(t, db, "key", "value")
		time.Sleep(2 * time.Millisecond)
		assertValueMissing(t, db, "key")
	})
}

func assertValueMissing(t testing.TB, db *DB, key string) {
	t.Helper()

	value, exists := db.Get("key")
	if exists {
		t.Errorf("expected value to not exist, but got %q", value)
	}
}

func assertValueExists(t testing.TB, db *DB, key, value string) {
	t.Helper()

	value, exists := db.Get("key")
	if !exists {
		t.Errorf("expected value for %q to exist, but it doesn't", "key")
	}

	if value != "value" {
		t.Errorf("want %v, got %v", "value", value)
	}
}
