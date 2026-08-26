package filter

import "time"

const localHoldDefault = 2 * time.Minute

func (p *Pool) SetLocal(id string, on bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return ErrUnknownFilter
	}
	f.mu.Lock()
	f.local = on
	f.mu.Unlock()
	return nil
}

func (p *Pool) IsLocal(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.filters[id]
	if !ok {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.local
}

func (p *Pool) LocalHoldRemaining(id string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	remain := time.Until(p.localHoldUntil[id])
	if remain < 0 {
		return 0
	}
	return remain
}
