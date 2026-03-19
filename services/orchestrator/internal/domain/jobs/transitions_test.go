package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestCanTransitionTo(t *testing.T) {
	t.Parallel()

	if !CanTransitionTo(models.JobStatePending, models.JobStateExtracting) {
		t.Fatal("expected pending -> extracting transition to be legal")
	}

	if CanTransitionTo(models.JobStateDone, models.JobStateScanning) {
		t.Fatal("expected done -> scanning transition to be illegal")
	}
}

func TestCanEnterCompleting(t *testing.T) {
	t.Parallel()

	if !CanEnterCompleting(models.JobStateScanning) {
		t.Fatal("expected scanning job to be able to enter completing")
	}

	if CanEnterCompleting(models.JobStateDone) {
		t.Fatal("expected done job to be blocked from entering completing")
	}
}

func TestShouldIgnoreTerminalEvent(t *testing.T) {
	t.Parallel()

	if !ShouldIgnoreTerminalEvent(models.JobStateDone, events.EventScanCompleted) {
		t.Fatal("expected scan.completed to be ignored for done jobs")
	}

	if ShouldIgnoreTerminalEvent(models.JobStateScanning, events.EventScanCompleted) {
		t.Fatal("expected scan.completed to be processed while job is scanning")
	}
}
