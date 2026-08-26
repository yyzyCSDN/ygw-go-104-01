package turb

type Thresholds struct {
	TurbidityHigh float64
	ChlorineLow   float64
}

type Decision struct {
	Dosing bool
}

type Assessor struct {
	thresholds Thresholds
}

func NewAssessor(thresholds Thresholds) *Assessor {
	return &Assessor{thresholds: thresholds}
}

func (a *Assessor) Evaluate(q Quality) Decision {
	decision := Decision{}
	if q.Turbidity >= a.thresholds.TurbidityHigh {
		decision.Dosing = true
	}
	if q.Chlorine <= a.thresholds.ChlorineLow {
		decision.Dosing = true
	}
	return decision
}
