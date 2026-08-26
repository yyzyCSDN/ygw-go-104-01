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

func TestBackwashWaitCancelledOnTimeout(t *testing.T) {
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
	p := pump.New("pump-1", pump.NewLocalDriver())
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
		100*time.Millisecond,
	)
	never := make(chan bool)
	if err := ctrl.Backwash("f1", never); !errors.Is(err, filter.ErrPressureTimeout) {
		t.Fatalf("expected pressure timeout, got %v", err)
	}
	if got := pool.States()["f1"]; got != filter.StateFiltering {
		t.Fatalf("filter must return to filtering after timeout, got %s", got)
	}
	if err := pool.BeginBackwash("f1"); err != nil {
		t.Fatalf("re-backwash must be allowed after timeout: %v", err)
	}
}
