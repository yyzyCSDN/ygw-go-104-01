package pump

import "sync"

type LocalDriver struct {
	mu     sync.Mutex
	active map[string]bool
}

func NewLocalDriver() *LocalDriver {
	return &LocalDriver{active: make(map[string]bool)}
}

func (d *LocalDriver) Activate(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.active[id] = true
	return nil
}

func (d *LocalDriver) Deactivate(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.active[id] = false
	return nil
}

func (d *LocalDriver) Active(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active[id]
}
