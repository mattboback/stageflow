package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/mattboback/stageflow/services/orchestrator/internal/orchtest"
)

var testDatabaseURL string

func TestMain(m *testing.M) {
	url, stop, err := orchtest.PostgresForTests("db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup test postgres: %v\n", err)
		os.Exit(1)
	}

	testDatabaseURL = url

	code := m.Run()

	if stopErr := stop(); stopErr != nil {
		fmt.Fprintf(os.Stderr, "stop test postgres: %v\n", stopErr)
	}

	os.Exit(code)
}
