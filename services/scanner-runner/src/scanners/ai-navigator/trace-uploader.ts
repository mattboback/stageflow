import fs from "fs-extra";
import path from "node:path";

import type { ScannerLogger, StorageProvider } from "../../core/types";

export async function uploadAiNavigatorTraces({
  storageProvider,
  bucket,
  prefix,
  resultsDir,
  logger,
}: {
  storageProvider: StorageProvider;
  bucket: string;
  prefix: string;
  resultsDir: string;
  logger: ScannerLogger;
}): Promise<void> {
  try {
    const entries = await fs.readdir(resultsDir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) {
        continue;
      }

      const tracePath = path.join(resultsDir, entry.name, "ai-trace.json");
      if (!(await fs.pathExists(tracePath))) {
        continue;
      }

      await storageProvider.upload(
        bucket,
        `${prefix}/${entry.name}/ai-trace.json`,
        tracePath,
      );
    }
  } catch (err) {
    logger.warn("Failed to upload ai-navigator traces", {
      error: err instanceof Error ? err.message : String(err),
    });
  }
}
