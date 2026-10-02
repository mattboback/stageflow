package main

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestScanWaitDefaultsCoverQueueAndExecution(t *testing.T) {
	for _, command := range []struct {
		name   string
		create func(*rootOptions) *cobra.Command
	}{
		{name: "scan", create: newScanCmd},
		{name: "live diff", create: newDiffCmd},
	} {
		t.Run(command.name, func(t *testing.T) {
			cmd := command.create(&rootOptions{})

			wait, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				t.Fatal(err)
			}

			if wait != 90*time.Minute {
				t.Fatalf("default wait = %v, want 90m for queue and execution", wait)
			}

			if parseErr := cmd.ParseFlags([]string{"--timeout", "17s"}); parseErr != nil {
				t.Fatal(parseErr)
			}

			wait, err = cmd.Flags().GetDuration("timeout")
			if err != nil {
				t.Fatal(err)
			}

			if wait != 17*time.Second {
				t.Fatalf("explicit wait = %v, want 17s", wait)
			}
		})
	}
}
