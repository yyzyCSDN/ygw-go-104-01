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

func TestStaleEventDoesNotOverrideNewState(t *testing.T) {
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
	critical := alarm.State{Level: alarm.LevelCritical, Subject: "inlet", Since: time.Now(), Value: 1.8, Metric: "turbidity"}
	restored := alarm.State{Level: alarm.LevelNormal, Subject: "inlet", Since: time.Now(), Value: 0.4, Metric: "turbidity"}
	if err := ctrl.HandleEvent(event.Event{Seq: 1, Kind: event.KindAlarm, Subject: "inlet", Payload: critical}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.HandleEvent(event.Event{Seq: 2, Kind: event.KindAlarm, Subject: "inlet", Payload: restored}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.HandleEvent(event.Event{Seq: 1, Kind: event.KindAlarm, Subject: "inlet", Payload: critical}); err != nil {
		t.Fatal(err)
	}
	if got := manager.State("inlet"); got != alarm.LevelNormal {
		t.Fatalf("stale event must not override newer state, got %s", got)
	}
}
