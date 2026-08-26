package turb

import (
	"encoding/json"
	"os"
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

func TestRecoveryRebuildsFromLiveFilterState(t *testing.T) {
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
	snapPath := filepath.Join(t.TempDir(), "snapshot.json")
	stale := Snapshot{
		SavedAt:      time.Now(),
		FilterStates: map[string]filter.State{"f1": filter.StateFault},
	}
	data, err := json.MarshalIndent(stale, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
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
		snapPath,
		time.Second,
	)
	if err := ctrl.Recover(); err != nil {
		t.Fatal(err)
	}
	if got := pool.States()["f1"]; got != filter.StateFiltering {
		t.Fatalf("recovery must keep the live filter state, got %s", got)
	}
}
