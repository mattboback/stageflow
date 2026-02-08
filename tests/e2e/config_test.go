package e2e

import "os"

var apiBaseURL = getAPIBaseURL()

func getAPIBaseURL() string {
	if v := os.Getenv("API_BASE_URL"); v != "" {
		return v
	}

	return "http://localhost:8080/api/v1"
}
