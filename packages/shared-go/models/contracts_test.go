package models_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestContractFixture_UnifiedReportV2_DecodeStrict(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "contracts", "report", "fixtures", "unified-report.v2.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var parsed report.UnifiedReportV2
	if err := dec.Decode(&parsed); err != nil {
		t.Fatalf("strict decode: %v", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			t.Fatalf("strict decode: unexpected trailing JSON")
		}
		t.Fatalf("strict decode: trailing content: %v", err)
	}

	if parsed.Version == "" || parsed.Meta.JobId == "" {
		t.Fatalf("unexpected report meta: %+v", parsed.Meta)
	}
	if len(parsed.Pages) == 0 || len(parsed.Issues) == 0 || len(parsed.Scanners) == 0 {
		t.Fatalf("expected scanners/pages/issues to be populated: scanners=%d pages=%d issues=%d",
			len(parsed.Scanners), len(parsed.Pages), len(parsed.Issues))
	}
}
