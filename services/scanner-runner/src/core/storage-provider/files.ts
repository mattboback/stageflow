import { stat } from "node:fs/promises";

import type { ScannerLogger } from "../types";

export async function waitForFileReady(
	filePath: string,
	logger: ScannerLogger,
	maxRetries = 5,
): Promise<boolean> {
	for (let attempt = 1; attempt <= maxRetries; attempt += 1) {
		try {
			const stats = await stat(filePath);
			if (stats.size > 0) {
				return true;
			}

			logger.debug("File is empty, retrying", {
				filePath,
				attempt,
				size: stats.size,
			});
		} catch (error) {
			logger.debug("File not ready", {
				filePath,
				attempt,
				error: error instanceof Error ? error.message : String(error),
			});
		}

		const delayMs = 100 * attempt;
		await new Promise((resolve) => setTimeout(resolve, delayMs));
	}

	return false;
}

export async function getFileSize(filePath: string): Promise<number> {
	try {
		const stats = await stat(filePath);
		return stats.size;
	} catch {
		return 0;
	}
}
