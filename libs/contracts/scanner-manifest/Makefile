.PHONY: all generate generate-ts generate-go validate clean

all: generate validate

generate:
	../../../devtools/scripts/generate-contracts.sh

generate-ts:
	../../../devtools/scripts/generate-contracts.sh ts

generate-go:
	../../../devtools/scripts/generate-contracts.sh go

validate:
	node schema/validate.js fixtures/scanner-manifest.min.json
	node schema/validate.js fixtures/scanner-manifest.full.json

clean:
	rm -rf generated scanner_manifest.go
