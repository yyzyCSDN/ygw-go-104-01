package alarm

import "errors"

var ErrUnknownAlarm = errors.New("unknown alarm subject")

func (m *Manager) Ack(subject string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.states[subject]
	if !ok {
		return ErrUnknownAlarm
	}
	st.Acked = true
	m.states[subject] = st
	return nil
}
