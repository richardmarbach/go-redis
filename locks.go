package redis

import "sync"

// A URWMutex is an upgradable RWMutex. Access to
// the RWMutex is guarded by a mutex.
type URWMutex struct {
	rw sync.RWMutex
	mu sync.Mutex
}

// RLock aquires the read lock
func (l *URWMutex) RLock() {
	l.mu.Lock()
	l.rw.RLock()
	l.mu.Unlock()
}

// RLock reliquishes the read lock
func (l *URWMutex) RUnlock() {
	l.mu.Lock()
	l.rw.RUnlock()
	l.mu.Unlock()
}

// Lock aquires the write lock
func (l *URWMutex) Lock() {
	l.mu.Lock()
	l.rw.Lock()
	l.mu.Unlock()
}

// Unlock reliquishes the write lock
func (l *URWMutex) Unlock() {
	l.mu.Lock()
	l.rw.Unlock()
	l.mu.Unlock()
}

// Upgrade a read lock into a write lock. A read lock should be aquired
// before calling this method.
func (l *URWMutex) Upgrade() {
	l.mu.Lock()
	l.rw.RUnlock()
	l.rw.Lock()
	l.mu.Unlock()
}

// Downgrade the write lock to a read lock. The write lock needs to be
// aquired before calling this method.
func (l *URWMutex) Downgrade() {
	l.mu.Lock()
	l.rw.Unlock()
	l.rw.RLock()
	l.mu.Unlock()
}
