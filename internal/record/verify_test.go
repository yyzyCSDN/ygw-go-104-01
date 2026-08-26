package record

import (
	"testing"
	"time"
)

func TestRecordHandleReleasedAfterWrite(t *testing.T) {
	rec, err := New(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		entry := Entry{
			At:        time.Now(),
			Source:    "inlet",
			Turbidity: 1.2,
			Chlorine:  0.4,
			Alarm:     "normal",
		}
		if err := rec.Append(entry); err != nil {
			t.Fatal(err)
		}
		if got := rec.OpenHandles(); got != 0 {
			t.Fatalf("record handle must be released after write, open=%d", got)
		}
	}
}
