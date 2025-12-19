package rateio

import (
	"sync"
	"testing"
	"time"
)

func TestKeyedLimiter_Basic(t *testing.T) {
	factory := func(key string) Limiter {
		return NewSimpleLimiter(10, time.Second)
	}
	grace := time.Minute
	kl := NewKeyedLimiter(factory, grace)

	lim1 := kl.Get("key1")
	if lim1 == nil {
		t.Fatal("Expected limiter for key1, got nil")
	}

	lim2 := kl.Get("key1")
	if lim1 != lim2 {
		t.Error("Expected same limiter instance for same key")
	}

	lim3 := kl.Get("key2")
	if lim3 == nil {
		t.Fatal("Expected limiter for key2, got nil")
	}
	if lim1 == lim3 {
		t.Error("Expected different limiter instances for different keys")
	}
}

func TestKeyedLimiter_GlobalExpiration(t *testing.T) {
	factory := func(key string) Limiter {
		return NewSimpleLimiter(10, time.Second)
	}
	grace := 50 * time.Millisecond
	kl := NewKeyedLimiter(factory, grace)

	kl.Get("key1")

	// Ensure key exists
	if _, ok := kl.limiters.Load("key1"); !ok {
		t.Fatal("key1 should exist")
	}

	// Sleep longer than grace
	time.Sleep(100 * time.Millisecond)

	// Next access should trigger global clear
	kl.Get("key2")

	// key1 should be gone
	if _, ok := kl.limiters.Load("key1"); ok {
		t.Error("key1 should have been cleared due to global expiration")
	}
	// key2 should exist
	if _, ok := kl.limiters.Load("key2"); !ok {
		t.Error("key2 should exist")
	}
}

func TestKeyedLimiter_EntryEviction(t *testing.T) {
	factory := func(key string) Limiter {
		return NewSimpleLimiter(10, time.Second)
	}
	grace := 100 * time.Millisecond
	kl := NewKeyedLimiter(factory, grace)
	// Set EvictAfterCount to small number for testing
	kl.EvictAfterCount = 2

	// Add key1
	kl.Get("key1")
	entry1Val, _ := kl.limiters.Load("key1")
	entry1 := entry1Val.(*keyedLimiterEntry)

	// Manually set expiration to the past
	entry1.expiresAt = time.Now().Add(-1 * time.Second).UnixNano()

	// Add key2 (count = 2), should trigger eviction
	// However, we need to make sure we don't access key1, because accessing it would renew it if we called Get("key1")
	// But we are calling Get("key2")
	kl.Get("key2")

	// Verify key1 is gone
	if _, ok := kl.limiters.Load("key1"); ok {
		t.Error("key1 should have been evicted")
	}
	// Verify key2 is present
	if _, ok := kl.limiters.Load("key2"); !ok {
		t.Error("key2 should be present")
	}
}

func TestKeyedLimiter_Concurrency(t *testing.T) {
	factory := func(key string) Limiter {
		return NewSimpleLimiter(10, time.Second)
	}
	grace := 500 * time.Millisecond
	kl := NewKeyedLimiter(factory, grace)
	kl.EvictAfterCount = 10

	var wg sync.WaitGroup
	// Spin up multiple goroutines accessing keys
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "common"
			if id%2 == 0 {
				key = "even"
			}
			kl.Get(key)
			time.Sleep(10 * time.Millisecond)
			kl.Get(key)
		}(i)
	}
	wg.Wait()
}
