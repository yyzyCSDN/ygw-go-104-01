package event

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrClosed = errors.New("event bus closed")

type Bus struct {
	mu        sync.Mutex
	seq       uint64
	order     chan Event
	subs      map[Kind][]chan Event
	closed    bool
	published atomic.Uint64
	delivered atomic.Uint64
}

func New() *Bus {
	return &Bus{
		order: make(chan Event, 256),
		subs:  make(map[Kind][]chan Event),
	}
}

func (b *Bus) Publish(kind Kind, subject string, payload any) (Event, error) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return Event{}, ErrClosed
	}
	b.seq++
	b.published.Add(1)
	ev := Event{
		Seq:     b.seq,
		Kind:    kind,
		Subject: subject,
		Payload: payload,
		At:      time.Now(),
	}
	b.mu.Unlock()
	b.order <- ev
	return ev, nil
}

func (b *Bus) Subscribe(kind Kind, size int) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, size)
	b.subs[kind] = append(b.subs[kind], ch)
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subs[kind]
		for i, item := range list {
			if item == ch {
				b.subs[kind] = append(list[:i], list[i+1:]...)
				break
			}
		}
		close(ch)
	}
}

func (b *Bus) Dispatch() {
	for ev := range b.order {
		b.mu.Lock()
		targets := b.subs[ev.Kind]
		b.mu.Unlock()
		for _, ch := range targets {
			select {
			case ch <- ev:
				b.delivered.Add(1)
			default:
			}
		}
	}
}

func (b *Bus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	subs := b.subs
	b.subs = nil
	b.mu.Unlock()
	close(b.order)
	for _, list := range subs {
		for _, ch := range list {
			close(ch)
		}
	}
}
