package main

import (
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestBuildProjectSubmitJobRequestDefaultsScreenshotCaptureOn(t *testing.T) {
	req, _, err := buildProjectSubmitJobRequest(projectmode.ScanConfig{
		URLs:     []string{"https://example.com"},
		Scanners: []string{"axe"},
	})
	testsupport.RequireNoErr(t, err)

	testsupport.RequireEqual(t, req.Screenshot, true, "req.Screenshot")
}

func TestBuildProjectSubmitJobRequestHonorsScreenshotFalse(t *testing.T) {
	disabled := false
	req, _, err := buildProjectSubmitJobRequest(projectmode.ScanConfig{
		URLs:       []string{"https://example.com"},
		Scanners:   []string{"axe"},
		Screenshot: &disabled,
	})
	testsupport.RequireNoErr(t, err)

	testsupport.RequireEqual(t, req.Screenshot, false, "req.Screenshot")
}
