package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	client := &http.Client{Timeout: 30 * time.Second}
	os.Exit(run(os.Args, os.Getenv, client, os.Stdout, os.Stderr))
}
