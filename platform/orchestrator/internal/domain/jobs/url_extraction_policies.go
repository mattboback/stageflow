package jobs

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

type URLJobPreparationAction string

const (
	URLJobPreparationAdvanceToReady URLJobPreparationAction = "advance_to_ready"
	URLJobPreparationAlreadyReady   URLJobPreparationAction = "already_ready"
	URLJobPreparationIgnore         URLJobPreparationAction = "ignore"
)

func DecideURLJobPreparation(state models.JobState) (URLJobPreparationAction, error) {
	switch state {
	case models.JobStateReady:
		return URLJobPreparationAlreadyReady, nil
	case models.JobStateScanning,
		models.JobStateCompleting,
		models.JobStateDone,
		models.JobStateFailed:
		return URLJobPreparationIgnore, nil
	case models.JobStatePending, models.JobStateExtracting:
		if !CanTransitionTo(state, models.JobStateReady) {
			return "", fmt.Errorf("job cannot transition to READY from %s", state)
		}

		return URLJobPreparationAdvanceToReady, nil
	default:
		return "", fmt.Errorf("job cannot transition to READY from %s", state)
	}
}

type ExtractionStartAction string

const (
	ExtractionStartAdvance           ExtractionStartAction = "advance"
	ExtractionStartAlreadyExtracting ExtractionStartAction = "already_extracting"
)

func DecideExtractionStart(state models.JobState) (ExtractionStartAction, error) {
	switch state {
	case models.JobStatePending:
		if !CanTransitionTo(state, models.JobStateExtracting) {
			return "", fmt.Errorf("job cannot transition to EXTRACTING from %s", state)
		}

		return ExtractionStartAdvance, nil
	case models.JobStateExtracting:
		return ExtractionStartAlreadyExtracting, nil
	case models.JobStateReady, models.JobStateScanning, models.JobStateCompleting,
		models.JobStateDone, models.JobStateFailed:
		return "", fmt.Errorf("job cannot transition to EXTRACTING from %s", state)
	default:
		return "", fmt.Errorf("job cannot transition to EXTRACTING from %s", state)
	}
}

func ValidateURLTargets(urls []string, allowsLoopbackTargets bool) error {
	if allowsLoopbackTargets || !containsLoopbackTargets(urls) {
		return nil
	}

	return errors.New("loopback targets require POD_NETNS_MODE=host for job pods (local dev only)")
}

func containsLoopbackTargets(urls []string) bool {
	for _, raw := range urls {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		u, err := url.Parse(trimmed)
		if err != nil {
			continue
		}

		host := u.Host
		if host == "" {
			continue
		}

		if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
			host = h
		}

		host = strings.Trim(host, "[]")

		if strings.EqualFold(host, "localhost") {
			return true
		}

		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return true
		}
	}

	return false
}
