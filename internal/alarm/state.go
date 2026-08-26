package alarm

import "time"

type Level string

const (
	LevelNormal   Level = "normal"
	LevelWarning  Level = "warning"
	LevelCritical Level = "critical"
)

type State struct {
	Level   Level
	Since   time.Time
	Value   float64
	Metric  string
	Subject string
	Acked   bool
}

func rank(level Level) int {
	switch level {
	case LevelCritical:
		return 3
	case LevelWarning:
		return 2
	case LevelNormal:
		return 1
	default:
		return 0
	}
}
