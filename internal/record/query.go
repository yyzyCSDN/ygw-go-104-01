package record

import (
	"bufio"
	"encoding/json"
	"os"
)

func (r *Recorder) Latest(source string) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, err := os.Open(r.path)
	if err != nil {
		return Entry{}, false
	}
	defer f.Close()
	var found Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Source == source {
			found = entry
		}
	}
	return found, found.Source == source
}

func (r *Recorder) FileSize() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := os.Stat(r.path)
	if err != nil {
		return 0
	}
	return info.Size()
}
