package pump

import (
	"fmt"
	"sync"
	"time"
)

type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StateFault   State = "fault"
)

type Driver interface {
	Activate(id string) error
	Deactivate(id string) error
	Active(id string) bool
}

type Pump struct {
	mu        sync.Mutex
	id        string
	driver    Driver
	state     State
	starts    int
	stops     int
	startedAt time.Time
}

func New(id string, driver Driver) *Pump {
	return &Pump{id: id, driver: driver, state: StateStopped}
}

func (p *Pump) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StateRunning {
		return nil
	}
	if err := p.driver.Activate(p.id); err != nil {
		return fmt.Errorf("pump %s activate: %w", p.id, err)
	}
	p.state = StateRunning
	p.starts++
	p.startedAt = time.Now()
	return nil
}

func (p *Pump) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StateStopped {
		return nil
	}
	if err := p.driver.Deactivate(p.id); err != nil {
		return fmt.Errorf("pump %s deactivate: %w", p.id, err)
	}
	p.state = StateStopped
	p.stops++
	p.startedAt = time.Time{}
	return nil
}

func (p *Pump) Check() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == StateRunning && !p.driver.Active(p.id) {
		p.state = StateFault
	}
	return p.state
}

func (p *Pump) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Pump) Starts() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.starts
}

func (p *Pump) Stops() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stops
}

func (p *Pump) ID() string {
	return p.id
}
