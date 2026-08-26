package filter

import "time"

type PressureSource func(id string) float64

func AutoConfirm(id string, source PressureSource, threshold float64, timeout time.Duration) <-chan bool {
	ch := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(timeout)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			if source(id) >= threshold {
				ch <- true
				return
			}
			if time.Now().After(deadline) {
				ch <- false
				return
			}
			select {
			case <-ticker.C:
			case <-time.After(time.Until(deadline)):
			}
		}
	}()
	return ch
}
