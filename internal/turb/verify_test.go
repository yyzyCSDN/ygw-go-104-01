package turb

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
	"waterplant/internal/record"
)

type failingDriver struct{}

func (failingDriver) Activate(id string) error {
	return errors.New("contactor stuck")
}

func (failingDriver) Deactivate(id string) error {
	return nil
}

func (failingDriver) Active(id string) bool {
	return false
}

func TestPumpStartFailureNotShownAsRunning(t *testing.T) {
	bus := event.New()
	journal := event.NewJournal(filepath.Join(t.TempDir(), "journal.jsonl"))
	dispatcher := alarm.NewDispatcher(bus, journal)
	manager := alarm.New(dispatcher)
	store := NewStore()
	assessor := NewAssessor(Thresholds{TurbidityHigh: 1.0, ChlorineLow: 0.3})
	pool := filter.NewPool("f1", "f2")
	recorder, err := record.New(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	p := pump.New("pump-1", failingDriver{})
	flow := dose.NewFlowWait(12.0, 0.5)
	plan := dose.NewPlan(0.5, 12.0, 0.08)
	limits := dose.NewLimits(0.05, 2.0)
	cooldown := dose.NewCooldown(0)
	doser := dose.New(p, flow, plan, func() float64 { return 12.0 }, 50*time.Millisecond)
	ctrl := NewController(
		store,
		assessor,
		pool,
		doser,
		manager,
		bus,
		recorder,
		journal,
		plan,
		limits,
		cooldown,
		filepath.Join(t.TempDir(), "snap.json"),
		time.Second,
	)
	cmd := dose.Command{ID: "cmd-1", Source: "inlet", Amount: 0.4, At: time.Now()}
	if err := ctrl.StartDosing(cmd); err == nil {
		t.Fatalf("pump start failure must propagate as an error")
	}
	if got := doser.Snapshot().State; got == dose.StateDosing {
		t.Fatalf("doser must not report dosing after pump start failure")
	}
	if got := p.State(); got == pump.StateRunning {
		t.Fatalf("pump must not report running after start failure")
	}
}
