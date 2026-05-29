package db

import (
	"errors"
	"strings"
	"testing"
)

type errOnlyJobRows struct {
	err error
}

func (r errOnlyJobRows) Next() bool { return false }
func (r errOnlyJobRows) Err() error { return r.err }
func (r errOnlyJobRows) Scan(...any) error {
	return errors.New("Scan should not be called")
}

func TestScanJobsReturnsRowsErr(t *testing.T) {
	t.Parallel()

	rowsErr := errors.New("stream interrupted")

	_, err := (&Database{}).scanJobs(errOnlyJobRows{err: rowsErr})
	if err == nil {
		t.Fatal("scanJobs() error = nil, want rows error")
	}

	if !strings.Contains(err.Error(), "failed to iterate jobs") || !errors.Is(err, rowsErr) {
		t.Fatalf("scanJobs() error = %v, want wrapped rows error", err)
	}
}
