package jobstore

import (
	"slices"

	"github.com/mattboback/stageflow/libs/go/models"
)

// stateRanks orders the job lifecycle. A job never moves to a lower rank, and
// the terminal states share the highest.
var stateRanks = map[models.JobState]int{
	models.JobStatePending:    0,
	models.JobStateExtracting: 1,
	models.JobStateReady:      2,
	models.JobStateScanning:   3,
	models.JobStateCompleting: 4,
	models.JobStateDone:       5,
	models.JobStateFailed:     5,
}

// stateRankSQL computes stateRanks in SQL, so an UPDATE can refuse to move a
// job backwards.
const stateRankSQL = `(CASE state
	WHEN 'PENDING' THEN 0
	WHEN 'EXTRACTING' THEN 1
	WHEN 'READY_TO_SCAN' THEN 2
	WHEN 'SCANNING' THEN 3
	WHEN 'COMPLETING' THEN 4
	ELSE 5
	END)`

// allowedTransitions is the single source of truth for the job lifecycle.
var allowedTransitions = map[models.JobState][]models.JobState{
	models.JobStatePending: {
		models.JobStateExtracting,
		models.JobStateReady,
		models.JobStateFailed,
	},
	models.JobStateExtracting: {
		models.JobStateReady,
		models.JobStateFailed,
	},
	models.JobStateReady: {
		models.JobStateScanning,
		models.JobStateFailed,
	},
	models.JobStateScanning: {
		models.JobStateCompleting,
		models.JobStateFailed,
	},
	models.JobStateCompleting: {
		models.JobStateDone,
		models.JobStateFailed,
	},
	models.JobStateDone:   {},
	models.JobStateFailed: {},
}

// stateRank returns state's lifecycle rank, or -1 for an unknown state.
func stateRank(state models.JobState) int {
	if rank, ok := stateRanks[state]; ok {
		return rank
	}

	return -1
}

// canTransition reports whether the lifecycle allows a job to move from one
// state to the other.
func canTransition(from, to models.JobState) bool {
	return slices.Contains(allowedTransitions[from], to)
}
