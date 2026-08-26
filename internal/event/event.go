package event

import "time"

type Kind string

const (
	KindQuality  Kind = "quality"
	KindBackwash Kind = "backwash"
	KindDose     Kind = "dose"
	KindAlarm    Kind = "alarm"
	KindLocal    Kind = "local"
	KindRecovery Kind = "recovery"
)

type Event struct {
	Seq     uint64
	Kind    Kind
	Subject string
	Payload any
	At      time.Time
}
