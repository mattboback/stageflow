# Schema Generation Makefile
#
# Usage:
#   make generate      - Generate both TypeScript and Go from schema
#   make generate-ts   - Generate TypeScript only
#   make generate-go   - Generate Go only
#   make validate      - Validate fixtures against schema
#   make clean         - Remove generated files
#   make install-tools - Install required generation tools

SCHEMA_FILE := schema/unified-report.v2.schema.json
TS_OUTPUT := generated/typescript/unified-report.v2.ts
GO_OUTPUT := generated/go/report_schema.go
GO_MOD_OUTPUT := generated/go/go.mod
GO_PACKAGE := report
GO_MODULE := github.com/mattboback/stageflow/libs/contracts/report/generated/go
GO_VERSION := 1.26.1

# Tool paths
GO_JSONSCHEMA := $(shell which go-jsonschema 2>/dev/null || echo "$(HOME)/go/bin/go-jsonschema")

.PHONY: all generate generate-ts generate-go validate clean install-tools

all: generate validate

generate: generate-ts generate-go
	@echo "✅ Generated TypeScript and Go from schema"

generate-ts: $(TS_OUTPUT)

$(TS_OUTPUT): $(SCHEMA_FILE)
	@mkdir -p $(dir $@)
	npx json2ts $(SCHEMA_FILE) -o $@ --unreachableDefinitions
	@echo "✅ Generated TypeScript: $@"

generate-go: $(GO_OUTPUT) $(GO_MOD_OUTPUT)

$(GO_OUTPUT): $(SCHEMA_FILE)
	@mkdir -p $(dir $@)
	$(GO_JSONSCHEMA) --package $(GO_PACKAGE) --tags json --struct-name-from-title $(SCHEMA_FILE) > $@
	goimports -w $@
	@echo "✅ Generated Go: $@"

$(GO_MOD_OUTPUT): Makefile
	@mkdir -p $(dir $@)
	@printf "module %s\n\ngo %s\n" "$(GO_MODULE)" "$(GO_VERSION)" > $@
	@echo "✅ Generated Go module: $@"

validate:
	@echo "Validating fixtures..."
	node schema/validate.js fixtures/unified-report.v2.json
	node schema/validate.js fixtures/unified-report.v2.all-scans.json
	@echo "✅ All fixtures valid"

clean:
	rm -rf generated/typescript/*.ts generated/go/*.go generated/go/go.mod
	@echo "✅ Cleaned generated files"

install-tools:
	bun add -d json-schema-to-typescript ajv ajv-formats
	go install github.com/atombender/go-jsonschema@v0.20.0
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "✅ Installed generation tools"

# Check if generated code is up to date
check-freshness:
	@make -q generate || (echo "❌ Generated code is out of date. Run 'make generate'" && exit 1)
	@echo "✅ Generated code is fresh"

# Diff generated Go vs existing types (TypeScript migration complete)
diff-go:
	@echo "=== Go differences (models/results.go vs generated) ==="
	@diff -u ../../libs/go/models/results.go $(GO_OUTPUT) || true
