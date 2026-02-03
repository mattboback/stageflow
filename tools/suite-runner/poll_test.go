package main

import "testing"

func TestEvaluate(t *testing.T) {
	maxCritical := 0
	maxSerious := 2
	maxTotal := 10

	th := Thresholds{
		MaxCritical: &maxCritical,
		MaxSerious:  &maxSerious,
		MaxTotal:    &maxTotal,
	}

	ok := evaluate(jobOutcome{Critical: 0, Serious: 2, TotalViolations: 10}, th)
	if !ok {
		t.Fatalf("expected outcome to pass thresholds")
	}

	failCritical := evaluate(jobOutcome{Critical: 1, Serious: 0, TotalViolations: 1}, th)
	if failCritical {
		t.Fatalf("expected critical threshold to fail")
	}

	failSerious := evaluate(jobOutcome{Critical: 0, Serious: 3, TotalViolations: 3}, th)
	if failSerious {
		t.Fatalf("expected serious threshold to fail")
	}

	failTotal := evaluate(jobOutcome{Critical: 0, Serious: 0, TotalViolations: 11}, th)
	if failTotal {
		t.Fatalf("expected total threshold to fail")
	}
}
