package rateio

import (
	"testing"
	"time"
)

func TestCount(t *testing.T) {
	limiter := NewSimpleLimiter(1, time.Millisecond)
	if err := limiter.Count(1); err != nil {
		t.Fatal("got rate limited on adding the first 'Count'")
	}

	if err := limiter.Count(1); err == nil {
		t.Fatal("expected to get rate limited on second 'Count'")
	}

}

func TestUntilNext(t *testing.T) {
	limiter := NewSimpleLimiter(1, time.Millisecond)
	if err := limiter.Count(1); err != nil {
		t.Fatal("got rate limited on adding the first entry")
	}

	next := limiter.UntilNext()
	<-next

	if err := limiter.Count(1); err != nil {
		t.Fatal("got rate limited on adding count after waiting")
	}
}

func TestNegativeUntilNext(t *testing.T) {
	limiter := NewSimpleLimiter(10, 10*time.Second)
	if err := limiter.Count(1); err != nil {
		t.Fatal("got rate limited on adding the first 'Count'")
	}

	before := time.Now()
	waitChan := limiter.UntilNext()
	after := <-waitChan

	if after.Sub(before) > 500*time.Millisecond {
		t.Fatal("waited too long, should have been immediate")
	}
}
