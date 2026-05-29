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

	want := []string{
		"ai-navigator",
		"axe",
		"lighthouse",
		"link-checker",
		"open-graph",
		"security-headers",
		"seo",
		"spelling-grammar",
	}

	if len(gotIDs) != len(want) {
		t.Fatalf("builtin manifest ids = %v, want %v", gotIDs, want)
	}

	for i := range want {
		if gotIDs[i] != want[i] {
			t.Fatalf("builtin manifest ids = %v, want %v", gotIDs, want)
		}
	}
}
