// Command job-status-cli queries the orchestrator admin API for job and pod details.
package main

import (
	"net/http"
	"os"
)

func main() {
	os.Exit(run(os.Args, os.Getenv, http.DefaultClient, os.Stdout, os.Stderr))
}
