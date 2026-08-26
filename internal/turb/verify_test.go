package turb

import (
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

func TestFlowTimeoutNotTreatedAsStable(t *testing.T) {
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
	doser := dose.New(p, flow, plan, func() float64 { return 0 }, 80*time.Millisecond)
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
		t.Fatalf("flow timeout must abort dosing")
	}
	if got := doser.Snapshot().State; got == dose.StateDosing {
		t.Fatalf("dosing must not start when flow is not stable")
	}
	if got := p.Starts(); got != 0 {
		t.Fatalf("pump must not start after flow timeout, got %d", got)
	}
}
