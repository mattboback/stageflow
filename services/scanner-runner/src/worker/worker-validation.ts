import type { ManifestConfigSchema } from "@stageflow/contracts-scanner-manifest";

import { validateOptionsAgainstSchema } from "../core/manifest";

export function assertScannerIdMatchesManifest(input: {
  manifestId: string;
  scannerId: string;
  strict: boolean;
  logger?: { warn(message: string, meta?: Record<string, unknown>): void };
}): void {
  if (input.scannerId === input.manifestId) {
    return;
  }

  const message = `Scanner metadata.name (${input.scannerId}) must match manifest.id (${input.manifestId})`;
  if (input.strict) {
    throw new Error(message);
  }

  input.logger?.warn(message, {
    manifestId: input.manifestId,
    scannerId: input.scannerId,
  });
}

export function assertScannerOptionsMatchSchema(input: {
  manifestId: string;
  schema: ManifestConfigSchema;
  options: unknown;
  strict: boolean;
  logger?: { warn(message: string, meta?: Record<string, unknown>): void };
}): void {
  // `SCANNER_OPTIONS` is optional for scanners whose manifest.configSchema doesn't require any fields.
  // Treat "missing" options as an empty object so object schemas validate successfully.
  const optionsForValidation = input.options === undefined ? {} : input.options;

  const validation = validateOptionsAgainstSchema(input.schema, optionsForValidation);
  if (validation.valid) {
    return;
  }

  const message = `SCANNER_OPTIONS did not match manifest.configSchema for '${input.manifestId}': ${validation.errors.join(
    "; ",
  )}`;

  if (input.strict) {
    throw new Error(message);
  }

  input.logger?.warn(message, {
    manifestId: input.manifestId,
    errorCount: validation.errors.length,
  });
}
