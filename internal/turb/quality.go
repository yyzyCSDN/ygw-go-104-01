package turb

import (
	"sync"
	"time"
)

const historyCapacity = 64

type Quality struct {
	ID        string
	Turbidity float64
	Chlorine  float64
	At        time.Time
}

type history struct {
	mu    sync.Mutex
	items []Quality
}

type Store struct {
	mu     sync.RWMutex
	latest map[string]Quality
	hist   map[string]*history
}

func NewStore() *Store {
	return &Store{
		latest: make(map[string]Quality),
		hist:   make(map[string]*history),
	}
}

func (s *Store) Put(q Quality) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Always store the latest reading for a source so dosing decisions act on
	// current values. The first-write-wins guard that used to live here dropped
	// every subsequent reading, leaving EvaluateDosing to judge stale data.
	s.latest[q.ID] = q
	h := s.hist[q.ID]
	if h == nil {
		h = &history{}
		s.hist[q.ID] = h
	}
	h.mu.Lock()
	h.items = append(h.items, q)
	if len(h.items) > historyCapacity {
		h.items = h.items[len(h.items)-historyCapacity:]
	}
	h.mu.Unlock()
}

func (s *Store) Latest(id string) (Quality, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.latest[id]
	return q, ok
}

func (s *Store) Snapshot() map[string]Quality {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Quality, len(s.latest))
	for id, q := range s.latest {
		out[id] = q
	}
	return out
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.latest)
}
