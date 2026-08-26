package turb

import (
	"testing"

	"waterplant/internal/filter"
)

func TestLocalWashStopsAutoBackwash(t *testing.T) {
	pool := filter.NewPool("f1", "f2")
	if err := pool.SetLocal("f1", true); err != nil {
		t.Fatal(err)
	}
	id, err := pool.NextForBackwash()
	if err != nil {
		t.Fatalf("expected an eligible filter, got %v", err)
	}
	if id == "f1" {
		t.Fatalf("auto rotation must not target a filter under local wash")
	}
}
