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

	previousTimeout := apiCommandHTTPTimeout
	apiCommandHTTPTimeout = 20 * time.Millisecond

	t.Cleanup(func() {
		apiCommandHTTPTimeout = previousTimeout
	})

	started := time.Now()
	stdout, stderr, exitCode := runCLI(t, "stageflow", "--api", server.URL, "scanners")
	elapsed := time.Since(started)

	close(block)
	server.Close()

	if exitCode != 2 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if elapsed > time.Second {
		t.Fatalf("scanners command took %s; expected bounded timeout", elapsed)
	}

	if !strings.Contains(stderr, "fetch scanners: failed to execute request") {
		t.Fatalf("expected request failure in stderr, got %q", stderr)
	}
}
