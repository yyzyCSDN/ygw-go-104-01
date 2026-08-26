package console

import (
	"errors"
	"time"

	"waterplant/internal/alarm"
	"waterplant/internal/dose"
	"waterplant/internal/event"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
	"waterplant/internal/record"
	"waterplant/internal/turb"
)

var ErrUnknownAction = errors.New("unknown console action")

type Console struct {
	ctrl     *turb.Controller
	pool     *filter.Pool
	doser    *dose.Doser
	pumps    *pump.Reporter
	alarms   *alarm.Manager
	store    *turb.Store
	recorder *record.Recorder
	bus      *event.Bus
}

func New(
	ctrl *turb.Controller,
	pool *filter.Pool,
	doser *dose.Doser,
	pumps *pump.Reporter,
	alarms *alarm.Manager,
	store *turb.Store,
	recorder *record.Recorder,
	bus *event.Bus,
) *Console {
	return &Console{
		ctrl:     ctrl,
		pool:     pool,
		doser:    doser,
		pumps:    pumps,
		alarms:   alarms,
		store:    store,
		recorder: recorder,
		bus:      bus,
	}
}

func (c *Console) Summary() Summary {
	return Summary{
		GeneratedAt:      time.Now(),
		Quality:          qualityViews(c.store),
		Filters:          filterViews(c.pool),
		Dosing:           c.doser.Snapshot(),
		Pumps:            c.pumps.Report(),
		Alarms:           c.alarms.Snapshot(),
		EventStats:       c.bus.Stats(),
		StaleEvents:      c.ctrl.StaleEvents(),
		Trends:           qualityTrends(c.store),
		Averages:         qualityAverages(c.store),
		AlarmCounts:      alarmCounts(c.alarms),
		NextBackwash:     nextBackwash(c.pool),
		ActiveBackwashes: c.pool.ActiveBackwashes(),
		AverageHeadLoss:  c.pool.AverageHeadLoss(),
		FilterCount:      c.pool.Count(),
		QualitySources:   c.store.Count(),
		PrimaryAlarm:     c.alarms.State("inlet"),
		LatestRecord:     latestRecord(c.recorder),
		RecordFileSize:   c.recorder.FileSize(),
		RecordsWritten:   c.recorder.Written(),
		RecordHandles:    c.recorder.OpenHandles(),
		RecordPath:       c.recorder.Path(),
	}
}

func nextBackwash(pool *filter.Pool) string {
	id, err := pool.NextForBackwash()
	if err != nil {
		return ""
	}
	return id
}

func latestRecord(recorder *record.Recorder) record.Entry {
	entry, _ := recorder.Latest("inlet")
	return entry
}

func (c *Console) Command(action, subject string) (string, error) {
	switch action {
	case "local_on":
		if err := c.pool.SetLocal(subject, true); err != nil {
			return "", err
		}
		_, _ = c.bus.Publish(event.KindLocal, subject, true)
		return "local-on", nil
	case "local_off":
		if err := c.pool.SetLocal(subject, false); err != nil {
			return "", err
		}
		_, _ = c.bus.Publish(event.KindLocal, subject, false)
		return "local-off", nil
	case "dose":
		if err := c.ctrl.EvaluateDosing(subject); err != nil {
			return "", err
		}
		return "dose-issued", nil
	case "dose_restart":
		// 投加泵启动失败后,值班员从这里重新下发启动,绕过冷却立即重试。
		if err := c.ctrl.RestartDosing(subject); err != nil {
			return "", err
		}
		return "dose-restarted", nil
	case "dose_stop":
		if err := c.doser.Stop("console"); err != nil {
			return "", err
		}
		return "dose-stopped", nil
	case "alarm_ack":
		if err := c.alarms.Ack(subject); err != nil {
			return "", err
		}
		return "alarm-acked", nil
	default:
		return "", ErrUnknownAction
	}
}
