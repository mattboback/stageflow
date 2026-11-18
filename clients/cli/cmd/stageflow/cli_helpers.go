package main

import "time"

// defaultScanWaitTimeout covers queue admission and the server's bounded job stages.
const defaultScanWaitTimeout = 90 * time.Minute

func envOr(getenv getenvFunc, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}
