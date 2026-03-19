package scannercatalog

import (
	"sort"
	"testing"
)

func TestBuiltinManifests(t *testing.T) {
	manifests, err := BuiltinManifests()
	if err != nil {
		t.Fatalf("BuiltinManifests() error: %v", err)
	}

	if len(manifests) == 0 {
		t.Fatal("BuiltinManifests() returned no manifests")
	}

	gotIDs := make([]string, 0, len(manifests))
	for _, m := range manifests {
		gotIDs = append(gotIDs, m.Id)
	}

	sort.Strings(gotIDs)

	want := []string{"ai-navigator", "axe", "lighthouse", "link-checker", "security-headers", "seo"}
	for _, id := range want {
		if i := sort.SearchStrings(gotIDs, id); i >= len(gotIDs) || gotIDs[i] != id {
			t.Fatalf("missing builtin manifest id %q; got %v", id, gotIDs)
		}
	}
}
