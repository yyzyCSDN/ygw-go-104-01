package filter

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrUnknownFilter   = errors.New("unknown filter")
	ErrDuplicateFilter = errors.New("duplicate filter")
	ErrBusy            = errors.New("filter busy")
	ErrNoCandidate     = errors.New("no filter candidate")
	ErrPressureTimeout = errors.New("pressure confirm timeout")
	ErrRejected        = errors.New("pressure confirm rejected")
)

type State string

const (
	StateFiltering   State = "filtering"
	StateBackwashing State = "backwashing"
	StateFault       State = "fault"
	StateLocal       State = "local"
)

type Record struct {
	ID    string
	Count int
	Last  time.Time
}

type Filter struct {
	mu    sync.Mutex
	ID    string
	State State
}

type Pool struct {
	mu             sync.Mutex
	filters        map[string]*Filter
	order          []string
	records        map[string]Record
	headloss       map[string]float64
	local          map[string]bool
	localHoldUntil map[string]time.Time
}

func NewPool(ids ...string) *Pool {
	pool := &Pool{
		filters:        make(map[string]*Filter, len(ids)),
		order:          append([]string(nil), ids...),
		records:        make(map[string]Record, len(ids)),
		headloss:       make(map[string]float64, len(ids)),
		local:          make(map[string]bool, len(ids)),
		localHoldUntil: make(map[string]time.Time, len(ids)),
	}
	for _, id := range ids {
		pool.filters[id] = &Filter{ID: id, State: StateFiltering}
	}
	return pool
}

func (p *Pool) Get(id string) (*Filter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	return f, ok
}

func (p *Pool) States() map[string]State {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]State, len(p.filters))
	for id, f := range p.filters {
		f.mu.Lock()
		out[id] = f.State
		f.mu.Unlock()
	}
	return out
}

func (p *Pool) SetState(id string, st State) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return ErrUnknownFilter
	}
	f.mu.Lock()
	f.State = st
	f.mu.Unlock()
	return nil
}

func (p *Pool) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.filters)
}

func (p *Pool) Register(f *Filter) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.filters[f.ID]; ok {
		return ErrDuplicateFilter
	}
	p.filters[f.ID] = f
	p.order = append(p.order, f.ID)
	return nil
}
