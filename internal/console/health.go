package console

import (
	"waterplant/internal/alarm"
	"waterplant/internal/filter"
	"waterplant/internal/pump"
)

func (c *Console) Health() string {
	if c.alarms.Count(alarm.LevelCritical) > 0 {
		return "critical"
	}
	for _, report := range c.pumps.Report() {
		if report.State == pump.StateFault {
			return "critical"
		}
	}
	for _, st := range c.pool.States() {
		if st == filter.StateFault {
			return "critical"
		}
	}
	if c.recorder.OpenHandles() > 8 {
		return "degraded"
	}
	if c.alarms.Count(alarm.LevelWarning) > 0 {
		return "degraded"
	}
	return "ok"
}
