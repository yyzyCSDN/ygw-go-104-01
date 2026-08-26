package pump

import "time"

func (p *Pump) RunSeconds() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != StateRunning {
		return 0
	}
	return int64(time.Since(p.startedAt).Seconds())
}
