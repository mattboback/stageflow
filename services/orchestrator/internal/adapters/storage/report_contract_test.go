package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
)

const reportContractRoot = "../../../../../"
const publicReportSchemaID = "https://stageflow.org/schemas/unified-report.v2.json"

func loadPublicReportSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	read := func(relative string) []byte {
		t.Helper()

		data, err := os.ReadFile(filepath.Join(reportContractRoot, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}

		return data
	}
	source := read("libs/contracts/report/schema/unified-report.v2.schema.json")

	public := read("clients/web/public/schemas/unified-report.v2.json")

	publicInfo, statErr := os.Stat(
		filepath.Join(reportContractRoot, "clients/web/public/schemas/unified-report.v2.json"),
	)
	if statErr != nil {
		t.Fatal(statErr)
	}

	if publicInfo.Mode().Perm()&0o044 != 0o044 {
		t.Fatalf(
			"public schema mode %o is unreadable to nginx workers; run just generate-contracts",
			publicInfo.Mode().Perm(),
		)
	}

	if !bytes.Equal(source, public) {
		t.Fatal("public schema differs from its source; run just generate-contracts")
	}

	var metadata struct {
		ID string `json:"$id"`
	}
	if err := json.Unmarshal(public, &metadata); err != nil {
		t.Fatal(err)
	}

	if metadata.ID != publicReportSchemaID {
		t.Fatalf("public schema $id=%q, want %q", metadata.ID, publicReportSchemaID)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(publicReportSchemaID, bytes.NewReader(public)); err != nil {
		t.Fatal(err)
	}

	schema, err := compiler.Compile(publicReportSchemaID)
	if err != nil {
		t.Fatal(err)
	}

	return schema
}

func validatePublicReport(t *testing.T, schema *jsonschema.Schema, data []byte) {
	t.Helper()

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}

	if err := schema.Validate(value); err != nil {
		t.Fatalf("public report schema validation: %v", err)
	}
}

func TestPublicReportSchemaAndDemoCatalog(t *testing.T) {
	t.Parallel()
	schema := loadPublicReportSchema(t)

	manifests, err := scannercatalog.BuiltinManifests()
	if err != nil {
		t.Fatal(err)
	}

	catalog := make(map[string]bool, len(manifests))
	for _, manifest := range manifests {
		catalog[manifest.Id] = true
	}

	for _, name := range []string{"baseline.json", "report.json"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, readErr := os.ReadFile(filepath.Join(reportContractRoot, "clients/web/public/demo", name))
			if readErr != nil {
				t.Fatal(readErr)
			}

			validatePublicReport(t, schema, data)

			var demo struct {
				Scanners []struct {
					ID string `json:"id"`
				} `json:"scanners"`
			}
			if decodeErr := json.Unmarshal(data, &demo); decodeErr != nil {
				t.Fatal(decodeErr)
			}

			for _, scanner := range demo.Scanners {
				if !catalog[scanner.ID] {
					t.Fatalf("demo references scanner absent from catalog: %s", scanner.ID)
				}
			}
		})
	}
}

func TestAggregatedReportValidatesPublicSchema(t *testing.T) {
	t.Parallel()
	schema := loadPublicReportSchema(t)

	fixture, err := os.ReadFile(
		filepath.Join(reportContractRoot, "libs/contracts/report/fixtures/unified-report.v2.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	mem := newMemoryFiles()
	mem.files["job-123/axe/results.json"] = fixture

	registry := scanners.NewRegistry("scanner-runner")
	if registerErr := registry.Register(
		&scanners.Definition{ID: "axe", Name: "Accessibility", Enabled: true},
	); registerErr != nil {
		t.Fatal(registerErr)
	}

	job := &models.Job{
		ID:               "job-123",
		ExpectedScanners: []string{"axe"},
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {ScannerType: "axe", ResultsPath: "job-123/axe/results.json", Success: true},
		},
	}

	key, err := NewAggregator(mem, registry).BuildAggregatedReport(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}

	validatePublicReport(t, schema, mem.files[key])
}
