.PHONY: all generate generate-ts generate-go validate clean

all: generate validate

generate generate-ts generate-go:
	../../../devtools/scripts/generate-contracts.sh

validate:
	node schema/validate.js fixtures/unified-report.v2.json
	node schema/validate.js fixtures/unified-report.v2.all-scans.json

clean:
	rm -rf generated
