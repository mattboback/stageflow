import fs from 'node:fs';
import { chromium } from 'playwright';

import { browserEnv } from '../../core/browser-env';
import type { ScannerLogger } from '../../core/types';
import type { LaunchedChrome } from './types';

// Lighthouse drives the Chromium that the runner image installs for Playwright.
export function resolveChromePath(): string {
	const playwrightPath = chromium.executablePath();
	if (playwrightPath && fs.existsSync(playwrightPath)) {
		return playwrightPath;
	}

	throw new Error('Unable to locate the Playwright Chromium executable for Lighthouse.');
}

export async function launchChromeForLighthouse(deps: {
	chromePath: string;
	logger: ScannerLogger;
}): Promise<LaunchedChrome> {
	const { chromePath, logger } = deps;

	const chromeLauncherModule = (await import('chrome-launcher')) as unknown as {
		launch?: (opts: Record<string, unknown>) => Promise<LaunchedChrome>;
		default?: {
			launch?: (opts: Record<string, unknown>) => Promise<LaunchedChrome>;
		};
	};

	const launch = chromeLauncherModule.launch ?? chromeLauncherModule.default?.launch;
	if (typeof launch !== 'function') {
		throw new Error('chrome-launcher launch() not available');
	}

	logger.info('Launching Chrome for Lighthouse', { chromePath });

	return launch({
		chromePath,
		envVars: browserEnv(),
		// Use a dedicated, headless Chrome process for Lighthouse.
		chromeFlags: [
			'--headless=new',
			'--no-sandbox',
			'--disable-setuid-sandbox',
			'--disable-dev-shm-usage',
			'--disable-gpu',
			'--no-first-run',
			'--no-default-browser-check',
			'--disable-background-networking',
			'--disable-background-timer-throttling',
			'--disable-renderer-backgrounding',
			'--disable-default-apps',
			'--disable-extensions',
			'--disable-sync',
			'--metrics-recording-only',
			'--mute-audio'
		]
	});
}

export async function closeLaunchedChrome(deps: {
	chrome: LaunchedChrome;
	logger: ScannerLogger;
}): Promise<void> {
	const { chrome, logger } = deps;

	try {
		await chrome.kill();
	} catch (error) {
		logger.warn('Failed to close Lighthouse Chrome', {
			error: error instanceof Error ? error.message : String(error)
		});
	}
}
