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

func TestDosingUsesFreshWaterQuality(t *testing.T) {
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
		time.Second,
	)
	now := time.Now()
	if err := ctrl.ApplyQuality(Quality{ID: "inlet", Turbidity: 0.4, Chlorine: 0.5, At: now}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.ApplyQuality(Quality{ID: "inlet", Turbidity: 1.8, Chlorine: 0.5, At: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.EvaluateDosing("inlet"); err != nil {
		t.Fatal(err)
	}
	if got := doser.Snapshot().Executed; got != 1 {
		t.Fatalf("dosing decision must use the fresh reading, executed=%d", got)
	}
}
