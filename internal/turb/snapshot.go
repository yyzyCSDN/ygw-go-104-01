package turb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"waterplant/internal/filter"
)

type Snapshot struct {
	SavedAt      time.Time
	FilterStates map[string]filter.State
	Qualities    map[string]Quality
}

func (c *Controller) SaveSnapshot() error {
	return c.saveSnapshotFrom(c.pool.States())
}

func (c *Controller) saveSnapshotFrom(states map[string]filter.State) error {
	snapshot := Snapshot{
		SavedAt:      time.Now(),
		FilterStates: states,
		Qualities:    c.store.Snapshot(),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.snapshotPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(c.snapshotPath, data, 0o644)
}

func (c *Controller) loadSnapshot() (Snapshot, error) {
	data, err := os.ReadFile(c.snapshotPath)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}
