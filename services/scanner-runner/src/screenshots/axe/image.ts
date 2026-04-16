import { join } from 'node:path';
import sharp from 'sharp';

import type { AxeScreenshotConfig, ElementBounds } from './types';

import { createLogger } from '../../utils/logger';

const log = createLogger('AxeImage');

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

/**
 * Composite highlight overlay rectangles onto a screenshot buffer using sharp.
 * This is CSP-safe as it doesn't require injecting CSS/JS into the page.
 *
 * @param clipCssWidth - The CSS pixel width of the clip region (used to compute device scale factor)
 */
export async function compositeOverlay(
	screenshotBuffer: Buffer,
	elementBounds: ElementBounds[],
	cfg: Pick<AxeScreenshotConfig, 'highlightStyle'>,
	clipCssWidth?: number
): Promise<Buffer> {
	if (elementBounds.length === 0) {
		return screenshotBuffer;
	}

	// Get image dimensions (in actual pixels, which may be scaled by device pixel ratio).
	const metadata = await sharp(screenshotBuffer).metadata();
	const width = metadata.width || 1280;
	const height = metadata.height || 720;

	// Compute the device scale factor from actual vs CSS dimensions.
	// Element bounds are in CSS pixels, but the image is in actual pixels.
	const dpr = clipCssWidth && clipCssWidth > 0 ? width / clipCssWidth : 1;

	// Generate SVG overlay with highlight rectangles.
	// Scale stroke width by DPR for consistent visual appearance.
	const strokeWidth = Math.round(6 * dpr);
	const outerStrokeWidth = Math.round(10 * dpr);

	// High-contrast outlines that remain visible on both light and dark backgrounds.
	const strokeColor = cfg.highlightStyle === 'dashed' ? '#ff2d55' : '#ff0000';
	const outerStrokeColor = 'rgba(0,0,0,0.85)';

	const fillColor = cfg.highlightStyle === 'dashed' ? 'rgba(255,45,85,0.12)' : 'rgba(255,0,0,0.12)';
	const dashArray =
		cfg.highlightStyle === 'dashed' ? `stroke-dasharray="${10 * dpr},${5 * dpr}"` : '';

	// Scale element bounds from CSS pixels to actual pixels.
	// We render a thicker dark outer stroke behind the colored stroke.
	const rects = elementBounds
		.map((box) => {
			const x = Math.round(box.x * dpr);
			const y = Math.round(box.y * dpr);
			const w = Math.round(box.width * dpr);
			const h = Math.round(box.height * dpr);

			return `
        <rect
          x="${x}"
          y="${y}"
          width="${w}"
          height="${h}"
          fill="none"
          stroke="${outerStrokeColor}"
          stroke-width="${outerStrokeWidth}"
        />
        <rect
          x="${x}"
          y="${y}"
          width="${w}"
          height="${h}"
          fill="${fillColor}"
          stroke="${strokeColor}"
          stroke-width="${strokeWidth}"
          ${dashArray}
        />`;
		})
		.join('\n');

	const svgOverlay = Buffer.from(`
      <svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">
        ${rects}
      </svg>
    `);

	// Composite overlay onto screenshot.
	return sharp(screenshotBuffer)
		.composite([{ input: svgOverlay, top: 0, left: 0 }])
		.toBuffer();
}

/**
 * Generate a thumbnail from a full-size screenshot.
 *
 * Returns the thumbnail filename or null if generation failed.
 */
export async function generateThumbnail(
	screenshotPath: string,
	resultsDir: string,
	cfg: Pick<
		AxeScreenshotConfig,
		'generateThumbnails' | 'thumbnailSize' | 'outputFormat' | 'webpQuality'
	>
): Promise<string | null> {
	if (!cfg.generateThumbnails) {
		return null;
	}

	try {
		const filename = screenshotPath.split('/').pop();
		if (!filename) {
			return null;
		}

		const ext = cfg.outputFormat === 'webp' ? '.webp' : '.png';
		const thumbnailFilename = filename.replace(/\.(png|webp)$/, `-thumb${ext}`);
		const thumbnailPath = join(resultsDir, thumbnailFilename);

		const pipeline = sharp(screenshotPath).resize(cfg.thumbnailSize, cfg.thumbnailSize, {
			fit: 'inside',
			withoutEnlargement: true
		});

		if (cfg.outputFormat === 'webp') {
			await pipeline.webp({ quality: cfg.webpQuality }).toFile(thumbnailPath);
		} else {
			await pipeline.png({ compressionLevel: 8 }).toFile(thumbnailPath);
		}

		return thumbnailFilename;
	} catch (error) {
		// Thumbnail generation is best-effort, don't fail the whole screenshot.
		log.warn('Failed to generate thumbnail', {
			screenshotPath,
			error: error instanceof Error ? error.message : String(error)
		});
		return null;
	}
}
