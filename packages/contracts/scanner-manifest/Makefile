# Scanner Manifest Schema Generation
#
# Usage:
#   make generate      - Generate both TypeScript and Go from schema
#   make generate-ts   - Generate TypeScript only
#   make generate-go   - Generate Go only
#   make validate      - Validate fixtures against schema
#   make clean         - Remove generated files
#   make install-tools - Install required generation tools

SCHEMA_FILE := schema/scanner-manifest.schema.json
TS_OUTPUT := generated/typescript/scanner-manifest.ts
GO_OUTPUT := scanner_manifest.go
GO_PACKAGE := scannermanifest

GO_JSONSCHEMA := $(shell which go-jsonschema 2>/dev/null || echo "$(HOME)/go/bin/go-jsonschema")
GOIMPORTS := $(shell which goimports 2>/dev/null || echo "$(HOME)/go/bin/goimports")

.PHONY: all generate generate-ts generate-go validate clean install-tools

all: generate validate

generate: generate-ts generate-go
	@echo "✅ Generated TypeScript and Go from schema"

generate-ts: $(TS_OUTPUT)

$(TS_OUTPUT): $(SCHEMA_FILE)
	@mkdir -p $(dir $@)
	npx json2ts $(SCHEMA_FILE) -o $@ --unreachableDefinitions
	@echo "✅ Generated TypeScript: $@"

generate-go: $(GO_OUTPUT)

$(GO_OUTPUT): $(SCHEMA_FILE)
	$(GO_JSONSCHEMA) --package $(GO_PACKAGE) --tags json --struct-name-from-title $(SCHEMA_FILE) > $@
	$(GOIMPORTS) -w $@
	@echo "✅ Generated Go: $@"

validate:
	@echo "Validating fixtures..."
	node schema/validate.js fixtures/scanner-manifest.min.json
	node schema/validate.js fixtures/scanner-manifest.full.json
	@echo "✅ All fixtures valid"

clean:
	rm -f generated/typescript/*.ts $(GO_OUTPUT)
	@echo "✅ Cleaned generated files"

install-tools:
	bun add -d json-schema-to-typescript ajv ajv-formats
	go install github.com/atombender/go-jsonschema@v0.20.0
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "✅ Installed generation tools"
