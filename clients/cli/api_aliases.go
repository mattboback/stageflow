package main

import (
	"net/http"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

type Client = apiclient.Client
type JobProgress = apiclient.JobProgress
type JobStatus = apiclient.JobStatus
type RemoteProject = apiclient.RemoteProject
type ScannerCapabilities = apiclient.ScannerCapabilities
type ScannerInfo = apiclient.ScannerInfo
type ScannersResponse = apiclient.ScannersResponse
type SubmitJobRequest = apiclient.SubmitJobRequest
type SubmitJobResponse = apiclient.SubmitJobResponse

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	return apiclient.NewClient(baseURL, apiKey, httpClient)
}
