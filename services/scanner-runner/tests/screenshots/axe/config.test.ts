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
		Reflect.deleteProperty(process.env, 'A11Y_HIGHLIGHT_STYLE');
		Reflect.deleteProperty(process.env, 'A11Y_SCREENSHOT_FORMAT');
		Reflect.deleteProperty(process.env, 'A11Y_OVERLAY_METHOD');
		Reflect.deleteProperty(process.env, 'A11Y_SEMANTIC_OVERLAY_ENABLED');
		Reflect.deleteProperty(process.env, 'A11Y_SEMANTIC_OVERLAY_MAX_HEADINGS');
		Reflect.deleteProperty(process.env, 'A11Y_SEMANTIC_OVERLAY_MAX_LABEL');
		Reflect.deleteProperty(process.env, 'A11Y_SEMANTIC_OVERLAY_LEGEND');

		const cfg = loadAxeScreenshotConfig();
		expect(cfg.highlightStyle).toBe('solid');
		expect(cfg.outputFormat).toBe('webp');
		expect(cfg.overlayMethod).toBe('sharp-composite');
		expect(cfg.semanticOverlayEnabled).toBe(true);
		expect(cfg.semanticOverlayMaxHeadings).toBe(30);
		expect(cfg.semanticOverlayMaxLabelLength).toBe(60);
		expect(cfg.semanticOverlayLegendEnabled).toBe(true);
	});

	it('parses known enums and clamps numeric values', () => {
		process.env.A11Y_HIGHLIGHT_STYLE = 'dashed';
		process.env.A11Y_SCREENSHOT_FORMAT = 'png';
		process.env.A11Y_OVERLAY_METHOD = 'css-injection';
		process.env.A11Y_SHOT_MAX_TARGETS = '-10';
		process.env.A11Y_SCREENSHOT_QUALITY = '999';
		process.env.A11Y_SEMANTIC_OVERLAY_MAX_HEADINGS = '-5';
		process.env.A11Y_SEMANTIC_OVERLAY_MAX_LABEL = '-9';
		process.env.A11Y_SEMANTIC_OVERLAY_ENABLED = 'false';
		process.env.A11Y_SEMANTIC_OVERLAY_LEGEND = 'false';

		const cfg = loadAxeScreenshotConfig();
		expect(cfg.highlightStyle).toBe('dashed');
		expect(cfg.outputFormat).toBe('png');
		expect(cfg.overlayMethod).toBe('css-injection');
		expect(cfg.unionMaxTargets).toBe(0);
		expect(cfg.webpQuality).toBe(100);
		expect(cfg.semanticOverlayEnabled).toBe(false);
		expect(cfg.semanticOverlayLegendEnabled).toBe(false);
		expect(cfg.semanticOverlayMaxHeadings).toBe(0);
		expect(cfg.semanticOverlayMaxLabelLength).toBe(0);
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
