package console

import (
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
	"waterplant/internal/record"
	"waterplant/internal/turb"
)

type Summary struct {
	GeneratedAt      time.Time
	Quality          map[string]QualityView
	Filters          map[string]FilterView
	Dosing           dose.Snapshot
	Pumps            []pump.Report
	Alarms           map[string]alarm.State
	EventStats       event.Stats
	StaleEvents      uint64
	Trends           map[string][]QualityView
	Averages         map[string]QualityView
	AlarmCounts      map[alarm.Level]int
	NextBackwash     string
	ActiveBackwashes int
	AverageHeadLoss  float64
	FilterCount      int
	QualitySources   int
	PrimaryAlarm     alarm.Level
	LatestRecord     record.Entry
	RecordFileSize   int64
	RecordsWritten   int
	RecordHandles    int
	RecordPath       string
}

type QualityView struct {
	Turbidity float64
	Chlorine  float64
	At        time.Time
}

type FilterView struct {
	State         filter.State
	Local         bool
	HoldRemaining time.Duration
	Backwashes    int
	HeadLoss      float64
	LastBackwash  time.Time
	BackwashDue   bool
}

func qualityViews(store *turb.Store) map[string]QualityView {
	out := make(map[string]QualityView)
	for id, q := range store.Snapshot() {
		out[id] = QualityView{Turbidity: q.Turbidity, Chlorine: q.Chlorine, At: q.At}
	}
	return out
}

func filterViews(pool *filter.Pool) map[string]FilterView {
	out := make(map[string]FilterView)
	for id, st := range pool.States() {
		rec, _ := pool.RecordFor(id)
		last, _ := pool.LastBackwash(id)
		out[id] = FilterView{
			State:         st,
			Local:         pool.IsLocal(id),
			HoldRemaining: pool.LocalHoldRemaining(id),
			Backwashes:    rec.Count,
			HeadLoss:      pool.HeadLoss(id),
			LastBackwash:  last,
			BackwashDue:   pool.BackwashDue(id, 5*time.Minute),
		}
	}
	return out
}

func qualityTrends(store *turb.Store) map[string][]QualityView {
	out := make(map[string][]QualityView)
	for id := range store.Snapshot() {
		recent := store.Recent(id, 3)
		if len(recent) == 0 {
			continue
		}
		views := make([]QualityView, 0, len(recent))
		for _, q := range recent {
			views = append(views, QualityView{Turbidity: q.Turbidity, Chlorine: q.Chlorine, At: q.At})
		}
		out[id] = views
	}
	return out
}

func qualityAverages(store *turb.Store) map[string]QualityView {
	out := make(map[string]QualityView)
	for id := range store.Snapshot() {
		q, ok := store.Average(id, 3)
		if !ok {
			continue
		}
		out[id] = QualityView{Turbidity: q.Turbidity, Chlorine: q.Chlorine, At: q.At}
	}
	return out
}

func alarmCounts(manager *alarm.Manager) map[alarm.Level]int {
	out := map[alarm.Level]int{
		alarm.LevelNormal:   manager.Count(alarm.LevelNormal),
		alarm.LevelWarning:  manager.Count(alarm.LevelWarning),
		alarm.LevelCritical: manager.Count(alarm.LevelCritical),
	}
	return out
}
