package cache

import (
	"testing"
	"time"
)

func TestPutAndGet(t *testing.T) {
	s := New()
	s.Put("key", "value", time.Minute)

	value, ok := s.Get("key")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if value != "value" {
		t.Fatalf("expected 'value', got %v", value)
	}
}

func TestGetMissing(t *testing.T) {
	s := New()

	_, ok := s.Get("missing")
	if ok {
		t.Fatal("expected missing key to not be found")
	}
}

func TestExpiry(t *testing.T) {
	s := New()
	s.Put("key", "value", time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, ok := s.Get("key")
	if ok {
		t.Fatal("expected expired key to not be found")
	}
}

func TestNoExpiry(t *testing.T) {
	s := New()
	s.Put("key", "value", 0)

	time.Sleep(5 * time.Millisecond)

	_, ok := s.Get("key")
	if !ok {
		t.Fatal("expected key without ttl to persist")
	}
}

func TestForget(t *testing.T) {
	s := New()
	s.Put("key", "value", time.Minute)
	s.Forget("key")

	_, ok := s.Get("key")
	if ok {
		t.Fatal("expected forgotten key to not be found")
	}
}
