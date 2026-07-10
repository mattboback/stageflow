#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GO_JSONSCHEMA="github.com/atombender/go-jsonschema@v0.20.0"
JSON_SCHEMA_TO_TYPESCRIPT="json-schema-to-typescript@15.0.4"

MODE="${1:-all}"
if [[ "$MODE" != "all" && "$MODE" != "ts" && "$MODE" != "go" ]]; then
  echo "Error: Unknown mode '$MODE'. Allowed modes: all, ts, go" >&2
  exit 1
fi

json2ts() {
  local schema="$1"
  local out="$2"

  mkdir -p "$(dirname "$out")"
  (cd "$ROOT_DIR" && bun x "$JSON_SCHEMA_TO_TYPESCRIPT" "$schema" -o "$out" --unreachableDefinitions)
}

generate_report() {
  local dir="$ROOT_DIR/libs/contracts/report"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/unified-report.v2.schema.json" "$dir/generated/typescript/unified-report.v2.ts"
    cat > "$dir/generated/typescript/index.ts" <<'EOF'
export * from './unified-report.v2';
export type { UnifiedReportV2 as UnifiedReport } from './unified-report.v2';
EOF
  fi

  if [[ "$MODE" == "all" || "$MODE" == "go" ]]; then
    mkdir -p "$dir/generated/go"
    go run "$GO_JSONSCHEMA" --package report --tags json --struct-name-from-title \
      "$dir/schema/unified-report.v2.schema.json" > "$dir/generated/go/report_schema.go"
    gofmt -w "$dir/generated/go/report_schema.go"
    printf "module github.com/mattboback/stageflow/libs/contracts/report/generated/go\n\ngo 1.26.5\n" \
      > "$dir/generated/go/go.mod"
  fi
}

generate_provenance() {
  local dir="$ROOT_DIR/libs/contracts/provenance"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/provenance.schema.json" "$dir/generated/typescript/provenance.ts"
    cat > "$dir/generated/typescript/index.ts" <<'EOF'
export * from './provenance';
EOF
  fi

  if [[ "$MODE" == "all" || "$MODE" == "go" ]]; then
    mkdir -p "$dir/generated/go"
    go run "$GO_JSONSCHEMA" --package provenance --tags json --struct-name-from-title \
      "$dir/schema/provenance.schema.json" > "$dir/generated/go/provenance_schema.go"
    gofmt -w "$dir/generated/go/provenance_schema.go"
    printf "module github.com/mattboback/stageflow/libs/contracts/provenance/generated/go\n\ngo 1.26.5\n" \
      > "$dir/generated/go/go.mod"
  fi
}

generate_scanner_manifest() {
  local dir="$ROOT_DIR/libs/contracts/scanner-manifest"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/scanner-manifest.schema.json" \
      "$dir/generated/typescript/scanner-manifest.ts"
    cat > "$dir/generated/typescript/index.ts" <<'EOF'
export * from './scanner-manifest';
EOF
  fi

  if [[ "$MODE" == "all" || "$MODE" == "go" ]]; then
    go run "$GO_JSONSCHEMA" --package scannermanifest --tags json --struct-name-from-title \
      "$dir/schema/scanner-manifest.schema.json" > "$dir/scanner_manifest.go"
    sed -i.bak \
      -e 's|^type ManifestConfigSchema interface{}$|// JSON Schema for SCANNER_OPTIONS validation (see ManifestConfigSchema definition\n// in schema/scanner-manifest.schema.json). Carried as json.RawMessage so the\n// Go side can forward the original bytes to the runtime validator.\ntype ManifestConfigSchema = json.RawMessage|' \
      -e 's|^type ScannerManifestConfigSchema interface{}$|// Embedded JSON Schema for the scanner-level configSchema field. Same\n// rationale as ManifestConfigSchema: round-tripped as raw JSON.\ntype ScannerManifestConfigSchema = json.RawMessage|' \
      "$dir/scanner_manifest.go"
    rm -f "$dir/scanner_manifest.go.bak"
    gofmt -w "$dir/scanner_manifest.go"
  fi
}

generate_report
generate_provenance
generate_scanner_manifest
