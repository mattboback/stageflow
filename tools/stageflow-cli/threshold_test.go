package main

import "testing"

func TestEvaluateThresholds(t *testing.T) {
	doc := sampleReport("job-threshold")

	t.Run("no thresholds", func(t *testing.T) {
		result := evaluateThresholds(doc, -1, -1, -1)

		if result.Evaluated {
			t.Fatal("expected thresholds to be skipped")
		}

		if !result.Passed {
			t.Fatal("expected skipped thresholds to pass")
		}
	})

	t.Run("critical fails first", func(t *testing.T) {
		result := evaluateThresholds(doc, 0, 10, 10)
		if result.Passed {
			t.Fatal("expected threshold failure")
		}

		if result.Detail != "critical: 1 > 0" {
			t.Fatalf("result.Detail = %q", result.Detail)
		}
	})

	t.Run("pass", func(t *testing.T) {
		result := evaluateThresholds(doc, 1, 0, 2)

		if !result.Passed {
			t.Fatalf("expected pass, got detail %q", result.Detail)
		}

		if result.Detail == "" {
			t.Fatal("expected threshold summary detail")
		}
	})
}
