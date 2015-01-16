package rateio

import (
	"strings"
	"testing"
	"time"
)

func TestReader(t *testing.T) {
	expected := "hello world"
	out := make([]byte, len(expected))

	b := NewReader(
		strings.NewReader(expected),
		NewSimpleLimiter(len(expected)-3, time.Second*1),
	)

	n, err := b.Read(out)
	if err != ErrRateExceeded {
		t.Error("Failed to exceed rate.")
	}

	b = NewReader(
		strings.NewReader(expected),
		NewSimpleLimiter(len(expected), time.Second*1),
	)
	n, err = b.Read(out)
	if err != nil {
		t.Error(err)
	}
	if n != len(expected) {
		t.Error("Read wrong amount.")
	}

	actual := string(out)
	if actual != expected {
		t.Errorf("Got: %v; Expected: %v", actual, expected)
	}
}
