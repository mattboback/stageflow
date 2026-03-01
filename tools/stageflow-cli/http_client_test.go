package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func withDefaultHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()

	prev := defaultHTTPClient
	defaultHTTPClient = client

	t.Cleanup(func() {
		defaultHTTPClient = prev
	})
}

func httpJSONResponse(t *testing.T, statusCode int, payload any) *http.Response {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	return httpResponse(statusCode, "application/json", append(data, '\n'))
}

func httpTextResponse(statusCode int, contentType string, body string) *http.Response {
	return httpResponse(statusCode, contentType, []byte(body))
}

func httpResponse(statusCode int, contentType string, body []byte) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	if contentType != "" {
		resp.Header.Set("Content-Type", contentType)
	}

	return resp
}
