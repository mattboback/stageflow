import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { loadAxeScreenshotConfig } from '../../../src/screenshots/axe/config';

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
}

describe('loadAxeScreenshotConfig', () => {
	beforeEach(() => {
		resetEnv();
	});
	afterEach(() => {
		resetEnv();
	});

	it('uses sensible defaults', () => {
		Reflect.deleteProperty(process.env, 'A11Y_SHOT_ENABLED');
		Reflect.deleteProperty(process.env, 'A11Y_SCREENSHOT_FORMAT');
		Reflect.deleteProperty(process.env, 'A11Y_SCREENSHOT_QUALITY');

		expect(loadAxeScreenshotConfig()).toEqual({
			screenshotsEnabled: true,
			outputFormat: 'webp',
			webpQuality: 80
		});
	});

	it('parses the format and clamps the quality', () => {
		process.env.A11Y_SHOT_ENABLED = 'false';
		process.env.A11Y_SCREENSHOT_FORMAT = 'PNG';
		process.env.A11Y_SCREENSHOT_QUALITY = '999';

		const cfg = loadAxeScreenshotConfig();
		expect(cfg.screenshotsEnabled).toBe(false);
		expect(cfg.outputFormat).toBe('png');
		expect(cfg.webpQuality).toBe(100);
	});

	it('applies overrides last', () => {
		const cfg = loadAxeScreenshotConfig({
			outputFormat: 'png',
			webpQuality: 10
		});
		expect(cfg.outputFormat).toBe('png');
		expect(cfg.webpQuality).toBe(10);
	});
});
