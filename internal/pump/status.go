package pump

import "sync"

type Report struct {
	ID         string
	State      State
	Starts     int
	Stops      int
	RunSeconds int64
}

type Reporter struct {
	mu    sync.Mutex
	pumps []*Pump
}

func NewReporter(pumps ...*Pump) *Reporter {
	return &Reporter{pumps: pumps}
}

func (r *Reporter) Report() []Report {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Report, 0, len(r.pumps))
	for _, p := range r.pumps {
		p.Check()
		out = append(out, Report{
			ID:         p.ID(),
			State:      p.State(),
			Starts:     p.Starts(),
			Stops:      p.Stops(),
			RunSeconds: p.RunSeconds(),
		})
	}
	return out
}
