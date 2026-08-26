package turb

import (
	"errors"
	"fmt"
	"os"

	"waterplant/internal/event"
	"waterplant/internal/filter"
)

func (c *Controller) Recover() error {
	prior := "fresh"
	snapshot, err := c.loadSnapshot()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && len(snapshot.FilterStates) > 0 {
		prior = fmt.Sprintf("registered %d snapshot filters", len(snapshot.FilterStates))
	}
	live := c.pool.States()
	for id, st := range snapshot.FilterStates {
		if _, ok := live[id]; ok {
			continue
		}
		if err := c.pool.Register(&filter.Filter{ID: id, State: st}); err != nil {
			return err
		}
	}
	if err := c.saveSnapshotFrom(c.pool.States()); err != nil {
		return err
	}
	events, err := c.journal.ReadAll()
	if err != nil {
		return err
	}
	for _, ev := range events {
		if err := c.HandleEvent(ev); err != nil {
			return err
		}
	}
	_, err = c.bus.Publish(event.KindRecovery, "system", prior)
	return err
}
