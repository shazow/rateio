package rateio

import (
	"sync"
	"sync/atomic"
	"time"
)

// KeyedLimiter manages a collection of Limiters, keyed by a string.
// It handles expiration of individual keys and the entire collection.
type KeyedLimiter struct {
	// Factory is the function to create a new Limiter for a given key.
	Factory func(string) Limiter

	// Grace is the duration after which a key (or the whole map) is considered stale.
	Grace time.Duration

	// EvictAfterCount configures how many additions trigger a stale entry cleanup.
	// Defaults to 100.
	EvictAfterCount int64

	limiters sync.Map

	mu         sync.Mutex
	lastAccess time.Time

	addCount int64
}

type keyedLimiterEntry struct {
	limiter   Limiter
	expiresAt int64 // UnixNano
}

// NewKeyedLimiter creates a new KeyedLimiter.
func NewKeyedLimiter(factory func(string) Limiter, grace time.Duration) *KeyedLimiter {
	return &KeyedLimiter{
		Factory:         factory,
		Grace:           grace,
		EvictAfterCount: 100,
		lastAccess:      time.Now(),
	}
}

// Get returns the Limiter for the given key.
// It handles expiration logic for the key and the global map.
func (kl *KeyedLimiter) Get(key string) Limiter {
	now := time.Now()

	// Check global expiration
	kl.mu.Lock()
	if now.Sub(kl.lastAccess) > kl.Grace {
		// Clear the whole map if global grace period exceeded
		// range over map to delete is not needed as sync.Map has Clear method since go 1.23,
		// but check go version support.
		// User environment has go 1.24.3, so Clear() is available.
		kl.limiters.Clear()
		// Optionally reset addCount, though not strictly required, it makes sense.
		atomic.StoreInt64(&kl.addCount, 0)
	}
	kl.lastAccess = now
	kl.mu.Unlock()

	// Prepare new entry in case we need it
	newExpiry := now.Add(kl.Grace).UnixNano()

	// We can't easily update the entry if we just pass a struct to LoadOrStore.
	// We need a pointer to the entry to update expiresAt atomically.
	newEntry := &keyedLimiterEntry{
		limiter:   kl.Factory(key),
		expiresAt: newExpiry,
	}

	val, loaded := kl.limiters.LoadOrStore(key, newEntry)

	entry := val.(*keyedLimiterEntry)

	if loaded {
		// Key exists, update expiration
		// We use atomic store to ensure safety if multiple goroutines access the same key
		atomic.StoreInt64(&entry.expiresAt, newExpiry)
	} else {
		// New key added
		count := atomic.AddInt64(&kl.addCount, 1)
		if kl.EvictAfterCount > 0 && count%kl.EvictAfterCount == 0 {
			kl.evictStale(now.UnixNano())
		}
	}

	return entry.limiter
}

func (kl *KeyedLimiter) evictStale(nowNano int64) {
	kl.limiters.Range(func(key, value any) bool {
		entry := value.(*keyedLimiterEntry)
		expiresAt := atomic.LoadInt64(&entry.expiresAt)
		if nowNano > expiresAt {
			kl.limiters.Delete(key)
		}
		return true
	})
}
