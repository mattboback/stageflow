package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestAllProjectDoctorChecksPassed_TrueWhenAllPass(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "passed"},
		{Name: "auth-valid", Status: "passed"},
	}

	if !allProjectDoctorChecksPassed(checks) {
		t.Fatal("expected true for all-passing checks, got false")
	}
}

func TestAllProjectDoctorChecksPassed_TrueForSkipped(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "passed"},
		{Name: "optional-check", Status: "skipped"},
	}

	if !allProjectDoctorChecksPassed(checks) {
		t.Fatal("expected true when only passed/skipped checks, got false")
	}
}

func TestAllProjectDoctorChecksPassed_TrueForEmptyChecks(t *testing.T) {
	if !allProjectDoctorChecksPassed(nil) {
		t.Fatal("expected true for empty check list, got false")
	}
}

func TestAllProjectDoctorChecksPassed_FalseWhenAnyFails(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "passed"},
		{Name: "auth-valid", Status: "failed", Message: "invalid API key"},
	}

	if allProjectDoctorChecksPassed(checks) {
		t.Fatal("expected false when a check has failed status, got true")
	}
}

func TestAllProjectDoctorChecksPassed_FalseForUnknownStatus(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "warning"},
	}

	if allProjectDoctorChecksPassed(checks) {
		t.Fatal("expected false for unknown/warning status, got true")
	}
}

func TestWriteProjectDoctorJSON_PassedTrueWhenAllChecksPass(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "passed"},
	}

	var out bytes.Buffer
	if err := writeProjectDoctorJSON(&out, "/repo", "/repo/.stageflow/config.yaml", "http://api", []string{"https://example.com"}, projectStageflowCfg{}, false, checks); err != nil {
		t.Fatalf("writeProjectDoctorJSON: %v", err)
	}

	var envelope projectDoctorEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !envelope.Passed {
		t.Error("expected Passed to be true when all checks pass")
	}
}

func TestWriteProjectDoctorJSON_PassedFalseWhenCheckFails(t *testing.T) {
	checks := []projectDoctorCheck{
		{Name: "api-reachable", Status: "failed", Message: "connection refused"},
	}

	var out bytes.Buffer
	if err := writeProjectDoctorJSON(&out, "/repo", "/repo/.stageflow/config.yaml", "http://api", []string{"https://example.com"}, projectStageflowCfg{}, false, checks); err != nil {
		t.Fatalf("writeProjectDoctorJSON: %v", err)
	}

	var envelope projectDoctorEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if envelope.Passed {
		t.Error("expected Passed to be false when a check fails")
	}
}

func TestWriteProjectDoctorJSON_SchemaIsSet(t *testing.T) {
	var out bytes.Buffer
	if err := writeProjectDoctorJSON(&out, "/repo", "/repo/.stageflow/config.yaml", "http://api", nil, projectStageflowCfg{}, false, nil); err != nil {
		t.Fatalf("writeProjectDoctorJSON: %v", err)
	}

	var envelope projectDoctorEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if envelope.Schema != "stageflow-cli/project-doctor@v1" {
		t.Errorf("expected schema 'stageflow-cli/project-doctor@v1', got %q", envelope.Schema)
	}
}
