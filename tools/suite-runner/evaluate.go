package main

func evaluate(o jobOutcome, th Thresholds) bool {
	if th.MaxCritical != nil && o.Critical > *th.MaxCritical {
		return false
	}

	if th.MaxSerious != nil && o.Serious > *th.MaxSerious {
		return false
	}

	if th.MaxTotal != nil && o.TotalViolations > *th.MaxTotal {
		return false
	}

	return true
}
