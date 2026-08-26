package alarm

type Operator string

const (
	OpAbove Operator = "above"
	OpBelow Operator = "below"
)

type Rule struct {
	Subject   string
	Metric    string
	Threshold float64
	Operator  Operator
	Level     Level
}

func Above(subject, metric string, threshold float64, level Level) Rule {
	return Rule{Subject: subject, Metric: metric, Threshold: threshold, Operator: OpAbove, Level: level}
}

func Below(subject, metric string, threshold float64, level Level) Rule {
	return Rule{Subject: subject, Metric: metric, Threshold: threshold, Operator: OpBelow, Level: level}
}

func (r Rule) Match(subject, metric string, value float64) bool {
	if r.Subject != "*" && r.Subject != subject {
		return false
	}
	if r.Metric != "*" && r.Metric != metric {
		return false
	}
	if r.Operator == OpAbove {
		return value >= r.Threshold
	}
	if r.Operator == OpBelow {
		return value <= r.Threshold
	}
	return false
}
