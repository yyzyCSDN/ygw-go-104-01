package dose

import (
	"errors"
	"math"
	"time"
)

var ErrFlowNotStable = errors.New("flow not stable within timeout")

type FlowWaiter interface {
	Wait(current func() float64, timeout time.Duration) error
}

type FlowWait struct {
	Stable    float64
	Tolerance float64
}

func NewFlowWait(stable, tolerance float64) *FlowWait {
	return &FlowWait{Stable: stable, Tolerance: tolerance}
}

func (w *FlowWait) Wait(current func() float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if time.Now().After(deadline) {
			return nil
		}
		if math.Abs(current()-w.Stable) <= w.Tolerance {
			return nil
		}
		select {
		case <-ticker.C:
		case <-time.After(time.Until(deadline)):
		}
	}
}
