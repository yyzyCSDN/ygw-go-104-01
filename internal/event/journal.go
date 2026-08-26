package event

import (
	"encoding/json"
	"os"
	"sync"
)

type Journal struct {
	mu   sync.Mutex
	path string
}

func NewJournal(path string) *Journal {
	return &Journal{path: path}
}

func (j *Journal) Append(ev Event) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	data, err := json.Marshal(ev)
	if err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (j *Journal) ReadAll() ([]Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	data, err := os.ReadFile(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []Event
	for len(data) > 0 {
		line := data
		if idx := indexByte(line, '\n'); idx >= 0 {
			line = line[:idx]
			data = data[idx+1:]
		} else {
			data = nil
		}
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (j *Journal) Truncate() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return os.Remove(j.path)
}

func (j *Journal) Size() int64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	info, err := os.Stat(j.path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func indexByte(data []byte, target byte) int {
	for i, b := range data {
		if b == target {
			return i
		}
	}
	return -1
}
