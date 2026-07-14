package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPICommandClient_TimesOutHungScannersResponse(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-block
	}))

	started := time.Now()
	client := newAPICommandClientWithTimeout(&rootOptions{apiURL: server.URL}, 20*time.Millisecond)

	var response any

	err := client.GetJSON(t.Context(), "/api/v1/scanners", &response)
	elapsed := time.Since(started)

	close(block)
	server.Close()

	if err == nil {
		t.Fatal("expected request timeout")
	}

	if elapsed > time.Second {
		t.Fatalf("scanners command took %s; expected bounded timeout", elapsed)
	}

	if !strings.Contains(err.Error(), "failed to execute request") {
		t.Fatalf("expected request failure, got %q", err)
	}
}
