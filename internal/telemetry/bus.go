// Package telemetry records rig events for operator console.
package telemetry

import (
	"sync"
	"time"
)

type Event struct {
	Source, Kind, RigID, Detail string
	ProcessTick                 int64
	At                          time.Time
}

type Ring struct {
	mu    sync.Mutex
	buf   []Event
	cap   int
	head  int
	count int
}

func NewRing(capacity int) *Ring {
	if capacity <= 0 {
		capacity = 256
	}
	return &Ring{buf: make([]Event, capacity), cap: capacity}
}

func (r *Ring) Push(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	r.buf[r.head] = ev
	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
}

func (r *Ring) Snapshot() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, r.count)
	if r.count == 0 {
		return out
	}
	start := r.head - r.count
	if start < 0 {
		start += r.cap
	}
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(start+i)%r.cap]
	}
	return out
}

type Bus struct {
	mu   sync.Mutex
	ring *Ring
}

func NewBus(capacity int) *Bus { return &Bus{ring: NewRing(capacity)} }

func (b *Bus) Emit(source, kind, rigID string, tick int64, detail string) {
	b.ring.Push(Event{Source: source, Kind: kind, RigID: rigID, Detail: detail, ProcessTick: tick, At: time.Now().UTC()})
}

func (b *Bus) Snapshot() []Event { return b.ring.Snapshot() }
