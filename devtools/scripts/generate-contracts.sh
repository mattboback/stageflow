#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GO_JSONSCHEMA="github.com/atombender/go-jsonschema@v0.20.0"
JSON_SCHEMA_TO_TYPESCRIPT="json-schema-to-typescript@15.0.4"
LOCK_DIR="$ROOT_DIR/.cache/generate-contracts.lock"

MODE="${1:-all}"
if [[ "$MODE" != "all" && "$MODE" != "ts" && "$MODE" != "go" ]]; then
  echo "Error: Unknown mode '$MODE'. Allowed modes: all, ts, go" >&2
  exit 1
fi

acquire_generation_lock() {
  mkdir -p "$(dirname "$LOCK_DIR")"

  local attempts=0
  until mkdir "$LOCK_DIR" 2>/dev/null; do
    attempts=$((attempts + 1))
    if (( attempts > 3000 )); then
      echo "Error: timed out waiting for contract generation lock: $LOCK_DIR" >&2
      exit 1
    fi
    sleep 0.1
  done

  trap 'rm -rf "$LOCK_DIR"' EXIT
}

atomic_temp_for() {
  local out="$1"
  mkdir -p "$(dirname "$out")"
  mktemp "${out}.tmp.XXXXXX"
}

write_atomic() {
  local out="$1"
  local temp
  temp="$(atomic_temp_for "$out")"
  cat > "$temp"
  mv -f "$temp" "$out"
}

acquire_generation_lock

json2ts() {
  local schema="$1"
  local out="$2"
  local temp

  temp="$(atomic_temp_for "$out")"
  (cd "$ROOT_DIR" && bun x "$JSON_SCHEMA_TO_TYPESCRIPT" "$schema" -o "$temp" --unreachableDefinitions)
  mv -f "$temp" "$out"
}

generate_report() {
  local dir="$ROOT_DIR/libs/contracts/report"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/unified-report.v2.schema.json" "$dir/generated/typescript/unified-report.v2.ts"
    write_atomic "$dir/generated/typescript/index.ts" <<'EOF'
export * from './unified-report.v2';
export type { UnifiedReportV2 as UnifiedReport } from './unified-report.v2';
EOF
  fi

  if [[ "$MODE" == "all" || "$MODE" == "go" ]]; then
    local report_go_temp
    report_go_temp="$(atomic_temp_for "$dir/generated/go/report_schema.go")"
    go run "$GO_JSONSCHEMA" --package report --tags json --struct-name-from-title \
      "$dir/schema/unified-report.v2.schema.json" > "$report_go_temp"
    gofmt -w "$report_go_temp"
    mv -f "$report_go_temp" "$dir/generated/go/report_schema.go"
    printf "module github.com/mattboback/stageflow/libs/contracts/report/generated/go\n\ngo 1.26.5\n" \
      | write_atomic "$dir/generated/go/go.mod"
  fi
}

generate_provenance() {
  local dir="$ROOT_DIR/libs/contracts/provenance"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/provenance.schema.json" "$dir/generated/typescript/provenance.ts"
    write_atomic "$dir/generated/typescript/index.ts" <<'EOF'
export * from './provenance';
EOF
  fi

  # No Go output. This used to emit a full module at
  # libs/contracts/provenance/generated/go, but go.work never included it and no
  # Go file ever imported it -- 18 generated types that were never compiled,
  # linted, or tested, regenerated in every job that runs this script. Go services
  # read provenance through the hand-written libs/go/models/provenance.go instead.
  #
  # Compare the two siblings that do earn their generation:
  # report/generated/go is in go.work with 38 importers, and scanner-manifest is a
  # real module consumed by libs/go/scannercatalog.
  #
  # Removing it loses no drift protection, because an unbuilt module provides
  # none. Getting that protection means making libs/go/models use these types and
  # adding the module to go.work -- a deliberate change, not a side effect of a
  # generator nobody reads.
}

generate_scanner_manifest() {
  local dir="$ROOT_DIR/libs/contracts/scanner-manifest"

  if [[ "$MODE" == "all" || "$MODE" == "ts" ]]; then
    json2ts "$dir/schema/scanner-manifest.schema.json" \
      "$dir/generated/typescript/scanner-manifest.ts"
    write_atomic "$dir/generated/typescript/index.ts" <<'EOF'
export * from './scanner-manifest';
EOF
  fi

  if [[ "$MODE" == "all" || "$MODE" == "go" ]]; then
    local manifest_go_temp
    manifest_go_temp="$(atomic_temp_for "$dir/scanner_manifest.go")"
    go run "$GO_JSONSCHEMA" --package scannermanifest --tags json --struct-name-from-title \
      "$dir/schema/scanner-manifest.schema.json" > "$manifest_go_temp"
    sed -i.bak \
      -e 's|^type ManifestConfigSchema interface{}$|// JSON Schema for SCANNER_OPTIONS validation (see ManifestConfigSchema definition\n// in schema/scanner-manifest.schema.json). Carried as json.RawMessage so the\n// Go side can forward the original bytes to the runtime validator.\ntype ManifestConfigSchema = json.RawMessage|' \
      -e 's|^type ScannerManifestConfigSchema interface{}$|// Embedded JSON Schema for the scanner-level configSchema field. Same\n// rationale as ManifestConfigSchema: round-tripped as raw JSON.\ntype ScannerManifestConfigSchema = json.RawMessage|' \
      "$manifest_go_temp"
    rm -f "$manifest_go_temp.bak"
    gofmt -w "$manifest_go_temp"
    mv -f "$manifest_go_temp" "$dir/scanner_manifest.go"
  fi
}

generate_report
generate_provenance
generate_scanner_manifest
