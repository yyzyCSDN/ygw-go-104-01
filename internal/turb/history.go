package turb

func (s *Store) Recent(id string, limit int) []Quality {
	s.mu.RLock()
	h, ok := s.hist[id]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	n := len(h.items)
	if limit > n {
		limit = n
	}
	out := make([]Quality, 0, limit)
	for i := n - limit; i < n; i++ {
		out = append(out, h.items[i])
	}
	return out
}

func (s *Store) Average(id string, window int) (Quality, bool) {
	recent := s.Recent(id, window)
	if len(recent) == 0 {
		return Quality{}, false
	}
	var q Quality
	q.ID = id
	q.At = recent[len(recent)-1].At
	for _, item := range recent {
		q.Turbidity += item.Turbidity
		q.Chlorine += item.Chlorine
	}
	q.Turbidity /= float64(len(recent))
	q.Chlorine /= float64(len(recent))
	return q, true
}
