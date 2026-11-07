package httpapi

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Now()
	l := NewLimiter(2, time.Minute)
	l.now = func() time.Time { return now }
	if !l.Allow("a") || !l.Allow("a") || l.Allow("a") {
		t.Fatal("expected the third call for the same key to be limited")
	}
	if !l.Allow("b") {
		t.Fatal("keys should be independent")
	}
	now = now.Add(time.Minute + time.Second)
	if !l.Allow("a") {
		t.Fatal("expected the window to reset")
	}
}
