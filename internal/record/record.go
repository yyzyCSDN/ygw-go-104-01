package record

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrCapacity = errors.New("record capacity reached")

type Entry struct {
	At        time.Time
	Source    string
	Turbidity float64
	Chlorine  float64
	Alarm     string
	Hash      string
}

type Recorder struct {
	mu      sync.Mutex
	dir     string
	max     int
	handles int
	written int
	path    string
}

func New(dir string, max int) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("record dir: %w", err)
	}
	return &Recorder{
		dir:  dir,
		max:  max,
		path: filepath.Join(dir, "records.jsonl"),
	}, nil
}

func (r *Recorder) Append(entry Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.written >= r.max {
		return ErrCapacity
	}
	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open record: %w", err)
	}
	r.handles++
	entry.Hash = Fingerprint(entry)
	line, err := json.Marshal(entry)
	if err != nil {
		_ = f.Close()
		r.handles--
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	if werr != nil {
		return werr
	}
	r.written++
	return nil
}

func (r *Recorder) OpenHandles() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.handles
}

func (r *Recorder) Written() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.written
}

func (r *Recorder) Path() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.path
}
