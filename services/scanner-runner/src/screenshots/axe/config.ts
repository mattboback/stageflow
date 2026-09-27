import type { AxeScreenshotConfig, OutputFormat } from './types';

import { getEnvBool, getEnvInt, getEnvString } from '../../utils/env';

function parseOutputFormat(raw: string): OutputFormat {
	return raw.toLowerCase() === 'png' ? 'png' : 'webp';
}

export function loadAxeScreenshotConfig(
	overrides?: Partial<AxeScreenshotConfig>
): AxeScreenshotConfig {
	return {
		screenshotsEnabled: getEnvBool('A11Y_SHOT_ENABLED', true),
		outputFormat: parseOutputFormat(getEnvString('A11Y_SCREENSHOT_FORMAT', 'webp')),
		webpQuality: Math.min(100, Math.max(1, getEnvInt('A11Y_SCREENSHOT_QUALITY', 80))),
		...overrides
	};
}
