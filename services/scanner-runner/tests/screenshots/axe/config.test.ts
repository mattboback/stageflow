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

		expect(loadAxeScreenshotConfig()).toEqual({
			screenshotsEnabled: true,
			outputFormat: 'webp',
			webpQuality: 80
		});
	});

	it('follows the job screenshot setting', () => {
		process.env.A11Y_SHOT_ENABLED = 'false';

		expect(loadAxeScreenshotConfig().screenshotsEnabled).toBe(false);
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
