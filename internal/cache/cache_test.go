package cache

import (
	"testing"
	"time"
)

func TestSetGetBeforeTTL(t *testing.T) {
	now := time.Unix(0, 0)
	c := New[int](func() time.Time { return now })
	c.Set("a", 42, 1000*time.Millisecond)
	got, ok := c.Get("a")
	if !ok || got != 42 {
		t.Errorf("want 42/true, got %d/%v", got, ok)
	}
}

func TestGetAfterTTL(t *testing.T) {
	now := time.Unix(0, 0)
	c := New[int](func() time.Time { return now })
	c.Set("a", 42, 1000*time.Millisecond)
	now = now.Add(1001 * time.Millisecond)
	_, ok := c.Get("a")
	if ok {
		t.Error("expected miss after TTL")
	}
}

func TestClear(t *testing.T) {
	now := time.Unix(0, 0)
	c := New[int](func() time.Time { return now })
	c.Set("a", 1, 10*time.Second)
	c.Set("b", 2, 10*time.Second)
	c.Clear()
	if _, ok := c.Get("a"); ok {
		t.Error("a should be cleared")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("b should be cleared")
	}
}

func TestOverwriteResetsTTL(t *testing.T) {
	now := time.Unix(0, 0)
	c := New[int](func() time.Time { return now })
	c.Set("a", 1, 1000*time.Millisecond)
	now = now.Add(500 * time.Millisecond)
	c.Set("a", 2, 1000*time.Millisecond)
	now = now.Add(700 * time.Millisecond)
	got, ok := c.Get("a")
	if !ok || got != 2 {
		t.Errorf("want 2/true, got %d/%v", got, ok)
	}
}
