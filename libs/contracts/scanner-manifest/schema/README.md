# Scanner Manifest Schema v1

## Overview

The Scanner Manifest schema defines the metadata and capabilities for StageFlow
scanner plugins. It is the single source of truth for manifest shape validation
across Go and TypeScript.

## Schema Location

- JSON Schema: `scanner-manifest.schema.json`
- Validation script: `validate.js`
- Fixtures: `../fixtures/`

## Key Concepts

- **id**: Stable scanner identifier (used in logs and artifact keys)
- **capabilities**: Scheduling and output characteristics
- **configSchema**: Optional JSON Schema for SCANNER_OPTIONS validation
- **entry**: Module entrypoint information for the scanner implementation

## Usage

### Validate a Manifest

```bash
node validate.js /path/to/manifest.json
```

### Validate Fixtures

```bash
node validate.js ../fixtures/scanner-manifest.min.json
node validate.js ../fixtures/scanner-manifest.full.json
```

## Notes

- `configSchema` accepts a JSON Schema object or boolean.
- When `supportsScreenshots` is true, `requiresBrowser` must also be true.
