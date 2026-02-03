package podman

import (
	"context"
	"net/http"
	"testing"
)

func TestRemoveVolume(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("DELETE", "/v4.0.0/libpod/volumes/test-volume", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force")
		if force != "true" {
			t.Errorf("expected force=true, got force=%s", force)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	if err := client.RemoveVolume(context.Background(), "test-volume", true); err != nil {
		t.Fatalf("RemoveVolume failed: %v", err)
	}
}
