package dose

import "errors"

var ErrInvalidPlan = errors.New("invalid dosing plan")

type Plan struct {
	TargetChlorine float64
	FlowRate       float64
	DoseRate       float64
}

func NewPlan(targetChlorine, flowRate, doseRate float64) *Plan {
	return &Plan{
		TargetChlorine: targetChlorine,
		FlowRate:       flowRate,
		DoseRate:       doseRate,
	}
}

func (p *Plan) AmountFor(chlorine float64) float64 {
	if chlorine >= p.TargetChlorine {
		return 0
	}
	return (p.TargetChlorine - chlorine) * p.FlowRate * p.DoseRate
}

func (p *Plan) Validate() error {
	if p.TargetChlorine <= 0 {
		return ErrInvalidPlan
	}
	if p.FlowRate <= 0 {
		return ErrInvalidPlan
	}
	if p.DoseRate <= 0 {
		return ErrInvalidPlan
	}
	return nil
}
