package filter

import "time"

const backwashMinInterval = 5 * time.Minute

func (p *Pool) SetHeadLoss(id string, value float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.filters[id]; !ok {
		return ErrUnknownFilter
	}
	p.headloss[id] = value
	return nil
}

func (p *Pool) HeadLoss(id string) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.headloss[id]
}

func (p *Pool) LastBackwash(id string) (time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.slot.ID == id {
		return p.slot.Last, true
	}
	return time.Time{}, false
}

func (p *Pool) BackwashDue(id string, interval time.Duration) bool {
	last, ok := p.LastBackwash(id)
	return !ok || time.Since(last) >= interval
}

func (p *Pool) AverageHeadLoss() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.filters) == 0 {
		return 0
	}
	var total float64
	for _, id := range p.order {
		total += p.headloss[id]
	}
	return total / float64(len(p.filters))
}

func (p *Pool) ActiveBackwashes() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	count := 0
	for _, f := range p.filters {
		f.mu.Lock()
		if f.State == StateBackwashing {
			count++
		}
		f.mu.Unlock()
	}
	return count
}
