package alarm

import (
	"time"

	"waterplant/internal/event"
)

type Dispatcher struct {
	bus     *event.Bus
	journal *event.Journal
}

func NewDispatcher(bus *event.Bus, journal *event.Journal) *Dispatcher {
	return &Dispatcher{bus: bus, journal: journal}
}

func (d *Dispatcher) Dispatch(level Level, subject, metric string, value float64) error {
	st := State{
		Level:   level,
		Since:   time.Now(),
		Value:   value,
		Metric:  metric,
		Subject: subject,
	}
	ev, err := d.bus.Publish(event.KindAlarm, subject, st)
	if err != nil {
		return err
	}
	return d.journal.Append(ev)
}
