package event

type Stats struct {
	Published   uint64
	Delivered   uint64
	Subscribers int
}

func (b *Bus) Stats() Stats {
	b.mu.Lock()
	subscribers := 0
	for _, list := range b.subs {
		subscribers += len(list)
	}
	b.mu.Unlock()
	return Stats{
		Published:   b.published.Load(),
		Delivered:   b.delivered.Load(),
		Subscribers: subscribers,
	}
}
