package api

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDecodeReportRetainsJobBoundaryValidation(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(buildStructuredScreenshotReport("job-1", time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}

	valid := string(encoded)
	for _, tt := range []struct {
		name, input string
		valid       bool
	}{
		{name: "matching job", input: valid, valid: true},
		{name: "different job", input: strings.ReplaceAll(valid, "job-1", "job-2")},
		{name: "missing job", input: `{}`},
		{name: "trailing document", input: valid + ` {}`},
		{name: "invalid JSON", input: `{"meta":`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, decodeErr := decodeReport([]byte(tt.input), "job-1")
			if tt.valid && decodeErr != nil {
				t.Fatalf("valid report rejected: %v", decodeErr)
			}

			if !tt.valid && !errors.Is(decodeErr, errInvalidReport) {
				t.Fatalf("expected invalid report, got %v", decodeErr)
			}
		})
	}
}
