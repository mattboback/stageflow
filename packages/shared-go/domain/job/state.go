// Package job provides domain logic for job state management.
package job

import (
	"slices"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// State ordering for comparison. Higher values represent later states in the lifecycle.
var stateOrder = map[models.JobState]int{
	models.JobStatePending:    0,
	models.JobStateExtracting: 1,
	models.JobStateReady:      2,
	models.JobStateScanning:   3,
	models.JobStateCompleting: 4,
	models.JobStateDone:       5,
	models.JobStateFailed:     5, // Terminal states share the highest rank
}

// allowedTransitions defines valid state transitions.
// This is the single source of truth for the job lifecycle.
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

// Order maps a state to a stable lifecycle rank; unknown states return -1.
func Order(state models.JobState) int {
	if rank, ok := stateOrder[state]; ok {
		return rank
	}

	return -1
}

// IsTerminal reports whether a state ends the lifecycle.
func IsTerminal(state models.JobState) bool {
	return state == models.JobStateDone || state == models.JobStateFailed
}

// IsLaterThan compares lifecycle ranks using Order.
func IsLaterThan(a, b models.JobState) bool {
	return Order(a) > Order(b)
}

// StateRankSQL returns a CASE expression mirroring Order for DB queries.
func StateRankSQL() string {
	return `(CASE state
		WHEN 'PENDING' THEN 0
	WHEN 'EXTRACTING' THEN 1
	WHEN 'READY_TO_SCAN' THEN 2
	WHEN 'SCANNING' THEN 3
	WHEN 'COMPLETING' THEN 4
	ELSE 5
	END)`
}

// CanTransition validates a transition against allowedTransitions.
func CanTransition(from, to models.JobState) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}

	return slices.Contains(allowed, to)
}

// AllowedTransitions exposes the lifecycle contract for callers.
func AllowedTransitions(from models.JobState) []models.JobState {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return []models.JobState{}
	}

	return allowed
}
