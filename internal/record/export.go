package record

import "os"

func (r *Recorder) Export(destination string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o644)
}
