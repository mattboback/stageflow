import { mkdtemp, readFile, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import sharp from 'sharp';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { saveScreenshot } from '../../../src/screenshots/axe/image';

async function redPng(): Promise<Buffer> {
	return sharp({
		create: { width: 8, height: 6, channels: 4, background: { r: 255, g: 0, b: 0, alpha: 1 } }
	})
		.png()
		.toBuffer();
}

describe('saveScreenshot', () => {
	let dir: string;

	beforeEach(async () => {
		dir = await mkdtemp(path.join(os.tmpdir(), 'stageflow-image-test-'));
	});

	afterEach(async () => {
		await rm(dir, { force: true, recursive: true });
	});

	it('writes WebP and returns the filename', async () => {
		const outputPath = path.join(dir, 'shot.webp');

		const filename = await saveScreenshot(await redPng(), outputPath, {
			outputFormat: 'webp',
			webpQuality: 80
		});

		expect(filename).toBe('shot.webp');
		const metadata = await sharp(await readFile(outputPath)).metadata();
		expect(metadata.format).toBe('webp');
		expect(metadata.width).toBe(8);
		expect(metadata.height).toBe(6);
	});

	it('writes PNG when configured', async () => {
		const outputPath = path.join(dir, 'shot.png');

		const filename = await saveScreenshot(await redPng(), outputPath, {
			outputFormat: 'png',
			webpQuality: 80
		});

		expect(filename).toBe('shot.png');
		const metadata = await sharp(await readFile(outputPath)).metadata();
		expect(metadata.format).toBe('png');
	});
});
