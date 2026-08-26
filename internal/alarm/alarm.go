package alarm

import (
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	mu         sync.Mutex
	rules      []Rule
	states     map[string]State
	dispatcher *Dispatcher
}

func New(dispatcher *Dispatcher, rules ...Rule) *Manager {
	return &Manager{
		rules:      rules,
		states:     make(map[string]State),
		dispatcher: dispatcher,
	}
}

func (m *Manager) Evaluate(subject, metric string, value float64) (Level, error) {
	var changed bool
	var next Level
	m.mu.Lock()
	next = m.evaluateRules(subject, metric, value)
	prev := m.states[subject].Level
	if next != prev {
		m.states[subject] = State{
			Level:   next,
			Since:   time.Now(),
			Value:   value,
			Metric:  metric,
			Subject: subject,
		}
		changed = true
	}
	m.mu.Unlock()
	if changed && m.dispatcher != nil {
		if err := m.dispatcher.Dispatch(next, subject, metric, value); err != nil {
			return next, fmt.Errorf("dispatch alarm: %w", err)
		}
	}
	return next, nil
}

func (m *Manager) evaluateRules(subject, metric string, value float64) Level {
	level := LevelNormal
	for _, rule := range m.rules {
		if rule.Match(subject, metric, value) && rank(rule.Level) > rank(level) {
			level = rule.Level
		}
	}
	return level
}

func (m *Manager) State(subject string) Level {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.states[subject].Level
}

func (m *Manager) Snapshot() map[string]State {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]State, len(m.states))
	for subject, st := range m.states {
		out[subject] = st
	}
	return out
}

func (m *Manager) ApplyEvent(subject string, st State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[subject] = st
	return nil
}

func (m *Manager) Count(level Level) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, st := range m.states {
		if st.Level == level {
			count++
		}
	}
	return count
}
