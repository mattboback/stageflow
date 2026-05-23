# Provenance Schema Generation
#
# Usage:
#   make generate      - Generate both TypeScript and Go from schema
#   make generate-ts   - Generate TypeScript only
#   make generate-go   - Generate Go only
#   make validate      - Validate fixtures against schema
#   make clean         - Remove generated files
#   make install-tools - Install required generation tools

SCHEMA_FILE := schema/provenance.schema.json
TS_OUTPUT := generated/typescript/provenance.ts
GO_OUTPUT := generated/go/provenance_schema.go
GO_MOD_OUTPUT := generated/go/go.mod
GO_PACKAGE := provenance
GO_MODULE := github.com/mattboback/stageflow/libs/contracts/provenance/generated/go
GO_VERSION := 1.26.3

GO_JSONSCHEMA := $(shell which go-jsonschema 2>/dev/null || echo "$(HOME)/go/bin/go-jsonschema")
GOIMPORTS := $(shell which goimports 2>/dev/null || echo "$(HOME)/go/bin/goimports")

.PHONY: all generate generate-ts generate-go validate clean install-tools

all: generate validate

generate: generate-ts generate-go
	@echo "Generated TypeScript and Go from schema"

generate-ts: $(TS_OUTPUT)

$(TS_OUTPUT): $(SCHEMA_FILE)
	@mkdir -p $(dir $@)
	npx json2ts $(SCHEMA_FILE) -o $@ --unreachableDefinitions
	@echo "Generated TypeScript: $@"

generate-go: $(GO_OUTPUT) $(GO_MOD_OUTPUT)

$(GO_OUTPUT): $(SCHEMA_FILE)
	@mkdir -p $(dir $@)
	$(GO_JSONSCHEMA) --package $(GO_PACKAGE) --tags json --struct-name-from-title $(SCHEMA_FILE) > $@
	$(GOIMPORTS) -w $@
	@echo "Generated Go: $@"

$(GO_MOD_OUTPUT): Makefile
	@mkdir -p $(dir $@)
	@printf "module %s\n\ngo %s\n" "$(GO_MODULE)" "$(GO_VERSION)" > $@
	@echo "Generated Go module: $@"

validate:
	@echo "Validating fixtures..."
	node schema/validate.js fixtures/provenance.no-auth.json
	node schema/validate.js fixtures/provenance.auth-storage-state.json
	node schema/validate.js fixtures/provenance.auth-form.json
	node schema/validate.js fixtures/provenance.auth-form-literal.json
	@echo "All fixtures valid"

clean:
	rm -rf generated/typescript/*.ts generated/go/*.go generated/go/go.mod
	@echo "Cleaned generated files"

install-tools:
	bun add -d json-schema-to-typescript ajv ajv-formats
	go install github.com/atombender/go-jsonschema@v0.20.0
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installed generation tools"
