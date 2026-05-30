package main

import (
	"net/http"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

var apiCommandHTTPTimeout = 30 * time.Second

func newAPICommandClient(root *rootOptions) *apiclient.Client {
	return apiclient.NewClient(root.apiURL, root.apiKey, &http.Client{
		Timeout: apiCommandHTTPTimeout,
	})
}
