import type { AxeScreenshotConfig } from './types';

import { getEnvBool } from '../../utils/env';

export function loadAxeScreenshotConfig(
	overrides?: Partial<AxeScreenshotConfig>
): AxeScreenshotConfig {
	return {
		// The orchestrator sets this from the job's screenshot option.
		screenshotsEnabled: getEnvBool('A11Y_SHOT_ENABLED', true),
		outputFormat: 'webp',
		webpQuality: 80,
		...overrides
	};
}
