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
	// 现场就地操作生效期间及之后一段保护期内，自动反冲洗调度让位：任何就地切换都开启
	// 一个保护窗口，避免自动策略紧接着对手洗过的滤池重复下发反冲。
	p.localHoldUntil[id] = time.Now().Add(localHoldDefault)
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
