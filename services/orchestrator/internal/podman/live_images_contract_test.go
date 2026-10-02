//go:build podmanlive

package podman

import (
	"context"
	"testing"
	"time"
)

// Job image names carry a registry host, a path and a tag, so the check must
// pass the full name through the Libpod route.
func TestLiveContract_ImageExistsReadsTheLocalStore(t *testing.T) {
	client := liveClient(t, liveSocketPath(t))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for name, want := range map[string]bool{
		liveImage(t): true,
		"localhost/stageflow/no-such-image:missing": false,
	} {
		exists, err := client.ImageExists(ctx, name)
		if err != nil || exists != want {
			t.Errorf("ImageExists(%q) = %v, %v; want %v", name, exists, err, want)
		}
	}
}
