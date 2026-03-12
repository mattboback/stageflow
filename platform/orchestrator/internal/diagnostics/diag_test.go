package diagnostics

import (
	"fmt"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/scannercatalog"
)

func TestPrintManifests(t *testing.T) {
	manifests, err := scannercatalog.BuiltinManifests()
	if err != nil {
		t.Fatalf("Failed to load manifests: %v", err)
	}

	fmt.Printf("COUNT: %d\n", len(manifests))

	for _, m := range manifests {
		fmt.Printf("SCANNER: %s\n", m.Id)
	}
}
