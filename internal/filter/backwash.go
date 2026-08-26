package filter

import (
	"time"
)

func (p *Pool) BeginBackwash(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return ErrUnknownFilter
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.State != StateFiltering {
		return ErrBusy
	}
	f.State = StateBackwashing
	return nil
}

func (p *Pool) ConfirmBackwash(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return ErrUnknownFilter
	}
	rec := p.slot
	rec.ID = id
	rec.Count++
	rec.Last = time.Now()
	p.slot = rec
	f.mu.Lock()
	f.State = StateFiltering
	f.mu.Unlock()
	return nil
}

func (p *Pool) CancelBackwash(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return ErrUnknownFilter
	}
	f.mu.Lock()
	f.State = StateFiltering
	f.mu.Unlock()
	return nil
}

func (p *Pool) RecordFor(id string) (Record, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.slot.ID == id {
		return p.slot, true
	}
	return Record{}, false
}
