package main

import (
	"net/http"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

const apiCommandHTTPTimeout = 30 * time.Second

func newAPICommandClient(root *rootOptions) *apiclient.Client {
	return newAPICommandClientWithTimeout(root, apiCommandHTTPTimeout)
}

func newAPICommandClientWithTimeout(root *rootOptions, timeout time.Duration) *apiclient.Client {
	return apiclient.NewClient(root.apiURL, root.apiKey, &http.Client{
		Timeout: timeout,
	})
}
