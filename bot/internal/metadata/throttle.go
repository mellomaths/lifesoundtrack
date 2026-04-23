package metadata

import (
	"sync"
	"time"
)

// mbThrottle enforces at least 1s between MusicBrainz requests (in-process, 1 rps).
type mbThrottle struct {
	mu  sync.Mutex
	min time.Duration
	// next is the earliest time a new MB request may start.
	next time.Time
}

func newMBThrottle() *mbThrottle {
	return &mbThrottle{min: time.Second}
}

// Wait blocks until 1rps is satisfied, then records this request.
func (t *mbThrottle) Wait() {
	if t == nil {
		return
	}
	now := time.Now()
	t.mu.Lock()
	if now.Before(t.next) {
		d := t.next.Sub(now)
		t.mu.Unlock()
		time.Sleep(d)
		t.mu.Lock()
		now = time.Now()
	}
	t.next = now.Add(t.min)
	t.mu.Unlock()
}
