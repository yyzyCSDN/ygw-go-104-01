package dose

import "time"

type State string

const (
	StateIdle   State = "idle"
	StateDosing State = "dosing"
	StateFault  State = "fault"
)

type Snapshot struct {
	State    State
	LastID   string
	Executed int
	Amount   float64
	Fault    string
	At       time.Time
}
