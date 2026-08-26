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
	p.local[id] = on
	f.mu.Lock()
	if on {
		f.State = StateLocal
	} else if f.State == StateLocal {
		f.State = StateFiltering
		p.localHoldUntil[id] = time.Now().Add(localHoldDefault)
	}
	f.mu.Unlock()
	return nil
}

func (p *Pool) IsLocal(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.local[id]
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
