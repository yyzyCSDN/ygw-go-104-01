package turb

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
	"waterplant/internal/record"
)

func TestFilterStateConcurrentNoOverwrite(t *testing.T) {
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
		2*time.Second,
	)
	confirmA := make(chan bool, 1)
	confirmB := make(chan bool, 1)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		<-start
		_ = ctrl.Backwash("f1", confirmA)
		wg.Done()
	}()
	go func() {
		<-start
		_ = ctrl.Backwash("f2", confirmB)
		wg.Done()
	}()
	close(start)
	confirmA <- true
	confirmB <- true
	wg.Wait()
	ra, okA := pool.RecordFor("f1")
	rb, okB := pool.RecordFor("f2")
	if !okA || !okB {
		t.Fatalf("each filter must keep its own backwash record: f1=%+v okA=%v f2=%+v okB=%v", ra, okA, rb, okB)
	}
	if ra.Count != 1 || rb.Count != 1 {
		t.Fatalf("each filter must count exactly one backwash: f1=%d f2=%d", ra.Count, rb.Count)
	}
}
