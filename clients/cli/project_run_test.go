package main

import "testing"

func TestBuildProjectSubmitJobRequestDefaultsScreenshotCaptureOn(t *testing.T) {
	req, _, err := buildProjectSubmitJobRequest(projectScanCfg{
		URLs:     []string{"https://example.com"},
		Scanners: []string{"axe"},
	})
	requireNoErr(t, err)

	requireEqual(t, req.Screenshot, true, "req.Screenshot")
}

func TestBuildProjectSubmitJobRequestHonorsScreenshotFalse(t *testing.T) {
	disabled := false
	req, _, err := buildProjectSubmitJobRequest(projectScanCfg{
		URLs:       []string{"https://example.com"},
		Scanners:   []string{"axe"},
		Screenshot: &disabled,
	})
	requireNoErr(t, err)

	requireEqual(t, req.Screenshot, false, "req.Screenshot")
}
