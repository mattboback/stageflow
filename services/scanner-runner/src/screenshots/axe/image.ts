import sharp from 'sharp';

import type { AxeScreenshotConfig } from './types';

/**
 * Save a screenshot buffer to disk in the configured format (WebP or PNG).
 * Returns the filename portion of the saved file.
 */
export async function saveScreenshot(
	buffer: Buffer,
	outputPath: string,
	cfg: Pick<AxeScreenshotConfig, 'outputFormat' | 'webpQuality'>
): Promise<string> {
	if (cfg.outputFormat === 'webp') {
		await sharp(buffer).webp({ quality: cfg.webpQuality }).toFile(outputPath);
	} else {
		await sharp(buffer).png({ compressionLevel: 8 }).toFile(outputPath);
	}

	return outputPath.split('/').pop() ?? outputPath;
}
