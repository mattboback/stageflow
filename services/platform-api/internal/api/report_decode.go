package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

var errInvalidReport = errors.New("report is invalid")

func decodeReport(data []byte, jobID string) (report.UnifiedReportV2, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var document report.UnifiedReportV2
	if err := decoder.Decode(&document); err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("%w: %w", errInvalidReport, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return report.UnifiedReportV2{}, fmt.Errorf("%w: trailing JSON document", errInvalidReport)
		}

		return report.UnifiedReportV2{}, fmt.Errorf("%w: trailing content: %w", errInvalidReport, err)
	}

	if document.Meta.JobId != jobID {
		return report.UnifiedReportV2{}, fmt.Errorf(
			"%w: report job ID %q does not match %q",
			errInvalidReport,
			document.Meta.JobId,
			jobID,
		)
	}

	return document, nil
}
