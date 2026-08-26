package dose

type Limits struct {
	Min float64
	Max float64
}

func NewLimits(min, max float64) *Limits {
	return &Limits{Min: min, Max: max}
}

func (l *Limits) Clamp(value float64) float64 {
	if value < l.Min {
		return l.Min
	}
	if value > l.Max {
		return l.Max
	}
	return value
}
