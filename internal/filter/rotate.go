package filter

import "time"

func (p *Pool) NextForBackwash() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	best := ""
	bestLoss := -1.0
	for _, id := range p.order {
		f := p.filters[id]
		f.mu.Lock()
		state := f.State
		f.mu.Unlock()
		if state != StateFiltering {
			continue
		}
		if p.local[id] {
			continue
		}
		if until := p.localHoldUntil[id]; time.Now().Before(until) {
			continue
		}
		rec := Record{}
		if p.slot.ID == id {
			rec = p.slot
		}
		if !rec.Last.IsZero() && time.Since(rec.Last) < backwashMinInterval {
			continue
		}
		loss := p.headloss[id]
		if loss > bestLoss {
			bestLoss = loss
			best = id
		}
	}
	if best == "" {
		return "", ErrNoCandidate
	}
	return best, nil
}
