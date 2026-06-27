.PHONY: all generate generate-ts generate-go validate clean

all: generate validate

generate generate-ts generate-go:
	../../../devtools/scripts/generate-contracts.sh

validate:
	node schema/validate.js fixtures/scanner-manifest.min.json
	node schema/validate.js fixtures/scanner-manifest.full.json

clean:
	rm -rf generated scanner_manifest.go
