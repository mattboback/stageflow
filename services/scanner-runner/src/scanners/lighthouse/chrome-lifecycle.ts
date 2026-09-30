import fs from 'node:fs';
import { chromium } from 'playwright';

import type { ScanStageLogger } from '../../core/scan-stage-logger';
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
	scanStageLogger: ScanStageLogger | null;
}): Promise<LaunchedChrome> {
	const { chromePath, logger, scanStageLogger } = deps;

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
	scanStageLogger?.recordEvent('lighthouse_chrome_launch', {
		chromePath
	});

	const chrome = await launch({
		chromePath,
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

	scanStageLogger?.recordEvent('lighthouse_chrome_ready', {
		port: chrome.port,
		pid: chrome.pid ?? null
	});

	return chrome;
}

export async function closeLaunchedChrome(deps: {
	chrome: LaunchedChrome;
	logger: ScannerLogger;
	scanStageLogger: ScanStageLogger | null;
}): Promise<void> {
	const { chrome, logger, scanStageLogger } = deps;

	try {
		await chrome.kill();
		scanStageLogger?.recordEvent('lighthouse_chrome_closed', {
			port: chrome.port,
			pid: chrome.pid ?? null
		});
	} catch (error) {
		logger.warn('Failed to close Lighthouse Chrome', {
			error: error instanceof Error ? error.message : String(error)
		});
	}
}
