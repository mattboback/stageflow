package api

import "time"

// Shared middleware constants. The middlewares themselves live in the
// middleware_*.go files, one per concern.

const (
	// defaultRequestTimeout is the maximum duration for processing a request.
	// This prevents slow clients or runaway handlers from consuming resources indefinitely.
	defaultRequestTimeout = 60 * time.Second

	// uploadRequestTimeout is extended for file upload endpoints.
	uploadRequestTimeout = 5 * time.Minute

	defaultRateLimitRequestsPerMinute = 120
	rateLimitWindow                   = time.Minute
	unknownRateLimitKey               = "unknown"
	rateLimiterMaxEntries             = 10000
	maxTimeoutResponseBytes           = 72 * 1024 * 1024

	defaultPublicSubmissionRequestsPerMinute = 6
	defaultPublicSubmissionBurst             = 3
	publicSubmissionRateEnv                  = "PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_RPM"
	publicSubmissionBurstEnv                 = "PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_BURST"
	defaultTrustedProxyCIDRs                 = "127.0.0.1/32,::1/128"
)
