package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/config"
	"waterplant/internal/console"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
	"waterplant/internal/record"
	"waterplant/internal/turb"
)

type flowState struct {
	mu    sync.Mutex
	value float64
}

func newFlowState(initial float64) *flowState {
	return &flowState{value: initial}
}

func (s *flowState) Set(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

func (s *flowState) Load() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

type pressureState struct {
	mu sync.RWMutex
	v  map[string]float64
}

func newPressureState(ids ...string) *pressureState {
	values := make(map[string]float64, len(ids))
	for _, id := range ids {
		values[id] = 0.12
	}
	return &pressureState{v: values}
}

func (s *pressureState) Set(id string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v[id] = value
}

func (s *pressureState) Load(id string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.v[id]
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func streamConsole(bus *event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		kinds := []event.Kind{
			event.KindQuality,
			event.KindBackwash,
			event.KindDose,
			event.KindAlarm,
			event.KindLocal,
			event.KindRecovery,
		}
		merged := make(chan event.Event, 128)
		for _, kind := range kinds {
			ch, unsub := bus.Subscribe(kind, 16)
			defer unsub()
			go func(src <-chan event.Event) {
				for ev := range src {
					merged <- ev
				}
			}(ch)
		}
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-merged:
				data, _ := json.Marshal(ev)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func main() {
	cfg, err := config.Load(os.Getenv("WPC_CONFIG"))
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	bus := event.New()
	go bus.Dispatch()
	defer bus.Close()

	journal := event.NewJournal(filepath.Join(cfg.DataDir, "journal.jsonl"))
	dispatcher := alarm.NewDispatcher(bus, journal)
	rules := []alarm.Rule{
		alarm.Above("*", "turbidity", cfg.TurbidityHigh, alarm.LevelCritical),
		alarm.Below("*", "chlorine", cfg.ChlorineLow, alarm.LevelWarning),
	}
	manager := alarm.New(dispatcher, rules...)

	driver := pump.NewLocalDriver()
	dosingPump := pump.New("dose-pump-1", driver)
	pumpReporter := pump.NewReporter(dosingPump)

	plan := dose.NewPlan(cfg.ChlorineLow+0.2, 12.0, 0.08)
	if err := plan.Validate(); err != nil {
		log.Fatalf("plan: %v", err)
	}
	limits := dose.NewLimits(0.05, 2.0)
	cooldown := dose.NewCooldown(cfg.DosingInterval() * 3)
	flow := newFlowState(plan.FlowRate)
	flowWait := dose.NewFlowWait(plan.FlowRate, 0.5)
	doser := dose.New(dosingPump, flowWait, plan, flow.Load, cfg.FlowTimeout())

	pool := filter.NewPool("f1", "f2", "f3", "f4")
	store := turb.NewStore()
	assessor := turb.NewAssessor(turb.Thresholds{
		TurbidityHigh: cfg.TurbidityHigh,
		ChlorineLow:   cfg.ChlorineLow,
	})
	recorder, err := record.New(filepath.Join(cfg.DataDir, "records"), cfg.MaxRecords)
	if err != nil {
		log.Fatalf("record: %v", err)
	}

	ctrl := turb.NewController(
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
		filepath.Join(cfg.DataDir, "snapshot.json"),
		cfg.BackwashTimeout(),
	)
	_ = ctrl.Recover()
	ctrl.Start()
	defer ctrl.Stop()

	pressure := newPressureState("f1", "f2", "f3", "f4")
	con := console.New(ctrl, pool, doser, pumpReporter, manager, store, recorder, bus)

	mux := http.NewServeMux()
	assets := http.FileServer(http.Dir("web"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join("web", "console.html"))
			return
		}
		assets.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/console/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, con.Summary())
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": con.Health()})
	})
	mux.HandleFunc("/api/quality", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID        string
			Turbidity float64
			Chlorine  float64
		}
		if !decodeBody(w, r, &req) {
			return
		}
		q := turb.Quality{ID: req.ID, Turbidity: req.Turbidity, Chlorine: req.Chlorine, At: time.Now()}
		if err := ctrl.ApplyQuality(q); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/dose", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Source string }
		if !decodeBody(w, r, &req) {
			return
		}
		if err := ctrl.EvaluateDosing(req.Source); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/filter/backwash", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID string }
		if !decodeBody(w, r, &req) {
			return
		}
		if _, ok := pool.Get(req.ID); !ok {
			http.Error(w, filter.ErrUnknownFilter.Error(), http.StatusNotFound)
			return
		}
		if pool.ActiveBackwashes() >= cfg.MaxConcurrentBackwash {
			http.Error(w, "too many concurrent backwashes", http.StatusConflict)
			return
		}
		confirm := filter.AutoConfirm(req.ID, pressure.Load, cfg.PressureThreshold, cfg.BackwashTimeout())
		if err := ctrl.Backwash(req.ID, confirm); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/filter/rotate", func(w http.ResponseWriter, r *http.Request) {
		id, err := pool.NextForBackwash()
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if pool.ActiveBackwashes() >= cfg.MaxConcurrentBackwash {
			http.Error(w, "too many concurrent backwashes", http.StatusConflict)
			return
		}
		confirm := filter.AutoConfirm(id, pressure.Load, cfg.PressureThreshold, cfg.BackwashTimeout())
		if err := ctrl.Backwash(id, confirm); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, map[string]string{"status": "ok", "filter": id})
	})
	mux.HandleFunc("/api/pressure", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID    string
			Value float64
		}
		if !decodeBody(w, r, &req) {
			return
		}
		pressure.Set(req.ID, req.Value)
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/flow", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Value float64 }
		if !decodeBody(w, r, &req) {
			return
		}
		flow.Set(req.Value)
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/headloss", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID    string
			Value float64
		}
		if !decodeBody(w, r, &req) {
			return
		}
		if err := pool.SetHeadLoss(req.ID, req.Value); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/filter/state", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID    string
			State string
		}
		if !decodeBody(w, r, &req) {
			return
		}
		if err := pool.SetState(req.ID, filter.State(req.State)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/console/command", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action  string
			Subject string
		}
		if !decodeBody(w, r, &req) {
			return
		}
		result, err := con.Command(req.Action, req.Subject)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"status": result})
	})
	mux.HandleFunc("/ws/console", streamConsole(bus))

	server := &http.Server{Addr: cfg.Addr, Handler: logRequests(mux)}
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = ctrl.SaveSnapshot()
	}()

	go func() {
		ticker := time.NewTicker(cfg.DosingInterval())
		defer ticker.Stop()
		for range ticker.C {
			_ = ctrl.EvaluateDosing("inlet")
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.RotateInterval())
		defer ticker.Stop()
		for range ticker.C {
			id, err := pool.NextForBackwash()
			if err != nil {
				continue
			}
			if pool.ActiveBackwashes() >= cfg.MaxConcurrentBackwash {
				continue
			}
			confirm := filter.AutoConfirm(id, pressure.Load, cfg.PressureThreshold, cfg.BackwashTimeout())
			_ = ctrl.Backwash(id, confirm)
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = ctrl.SaveSnapshot()
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if journal.Size() > 1<<20 {
				_ = journal.Truncate()
			}
			if recorder.Written() > 5000 {
				_ = recorder.Export(filepath.Join(cfg.DataDir, "records", "export-"+time.Now().Format("20060102")+".jsonl"))
				_ = recorder.Archive()
			}
		}
	}()

	log.Printf("water plant control listening on %s", cfg.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
