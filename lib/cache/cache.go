// Package cache provides a lightweight in-memory TTL store used for the
// ephemeral ACS state (lookup results, one-shot provision flags) that PHP
// kept in Redis. Not persisted, not shared across processes - fine for a
// single-instance ACS server.
package cache

import (
	"sync"
	"time"
)

type item struct {
	value     interface{}
	expiresAt time.Time
	hasExpiry bool
}

type Store struct {
	mu    sync.RWMutex
	items map[string]item
}

func New() *Store {
	return &Store{items: make(map[string]item)}
}

// Global is the process-wide cache instance, initialized at startup in main.go.
var Global = New()

func (s *Store) Put(key string, value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := item{value: value}
	if ttl > 0 {
		entry.hasExpiry = true
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.items[key] = entry
}

func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	entry, ok := s.items[key]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if entry.hasExpiry && time.Now().After(entry.expiresAt) {
		s.Forget(key)
		return nil, false
	}

	return entry.value, true
}

func (s *Store) Forget(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// StartJanitor periodically purges expired entries so long-lived TTL misses
// don't accumulate in memory. Call once at startup as a goroutine.
func (s *Store) StartJanitor(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			s.purgeExpired()
		}
	}()
}

func (s *Store) purgeExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, entry := range s.items {
		if entry.hasExpiry && now.After(entry.expiresAt) {
			delete(s.items, key)
		}
	}
}
