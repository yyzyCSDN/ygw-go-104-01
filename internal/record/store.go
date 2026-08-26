package record

import (
	"os"
	"path/filepath"
	"time"
)

func (r *Recorder) Archive() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.written == 0 {
		return nil
	}
	name := "records-" + time.Now().Format("20060102-150405") + ".jsonl"
	dst := filepath.Join(r.dir, name)
	if err := os.Rename(r.path, dst); err != nil {
		return err
	}
	r.written = 0
	return nil
}
