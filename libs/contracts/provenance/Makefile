.PHONY: all generate generate-ts generate-go validate clean

all: generate validate

generate generate-ts generate-go:
	../../../devtools/scripts/generate-contracts.sh

validate:
	node schema/validate.js fixtures/provenance.no-auth.json
	node schema/validate.js fixtures/provenance.auth-storage-state.json
	node schema/validate.js fixtures/provenance.auth-form.json
	node schema/validate.js fixtures/provenance.auth-form-literal.json

clean:
	rm -rf generated
