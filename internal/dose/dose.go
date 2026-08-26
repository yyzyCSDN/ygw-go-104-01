package dose

import (
	"fmt"
	"sync"
	"time"

	"waterplant/internal/pump"
)

type Command struct {
	ID     string
	Source string
	Amount float64
	At     time.Time
}

type Doser struct {
	mu          sync.Mutex
	pump        *pump.Pump
	flow        FlowWaiter
	plan        *Plan
	flowFn      func() float64
	flowTimeout time.Duration
	state       State
	executed    map[string]bool
	count       int
	last        Command
	fault       string
}

func New(pump *pump.Pump, flow FlowWaiter, plan *Plan, flowFn func() float64, flowTimeout time.Duration) *Doser {
	return &Doser{
		pump:        pump,
		flow:        flow,
		plan:        plan,
		flowFn:      flowFn,
		flowTimeout: flowTimeout,
		state:       StateIdle,
		executed:    make(map[string]bool),
	}
}

func (d *Doser) Start(cmd Command) error {
	d.mu.Lock()
	if d.executed[cmd.ID] {
		d.mu.Unlock()
		return nil
	}
	d.mu.Unlock()
	if err := d.flow.Wait(d.flowFn, d.flowTimeout); err != nil {
		return fmt.Errorf("flow wait: %w", err)
	}
	_ = d.pump.Start()
	d.mu.Lock()
	d.executed[cmd.ID] = true
	d.count++
	d.last = cmd
	d.state = StateDosing
	d.mu.Unlock()
	return nil
}

func (d *Doser) Stop(reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.state != StateDosing {
		return nil
	}
	if err := d.pump.Stop(); err != nil {
		d.state = StateFault
		d.fault = reason + ": " + err.Error()
		return err
	}
	d.state = StateIdle
	d.fault = ""
	return nil
}

func (d *Doser) Snapshot() Snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return Snapshot{
		State:    d.state,
		LastID:   d.last.ID,
		Executed: d.count,
		Amount:   d.last.Amount,
		Fault:    d.fault,
		At:       d.last.At,
	}
}
