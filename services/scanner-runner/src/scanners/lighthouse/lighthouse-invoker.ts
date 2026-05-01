import type { ScanStageLogger } from '../../core/scan-stage-logger';
import type { ScannerLogger } from '../../core/types';
import type { LighthouseModule, LighthouseResult } from './types';

export function loadLighthouseModule(): Promise<LighthouseModule> {
	return import('lighthouse');
}

export async function runLighthouseInvocation(deps: {
	url: string;
	port: number;
	categories: string[];
	lighthouse: LighthouseModule['default'];
	logger: ScannerLogger;
	scanStageLogger: ScanStageLogger | null;
}): Promise<LighthouseResult> {
	const { url, port, categories, lighthouse, logger, scanStageLogger } = deps;

	logger.info('Running Lighthouse', {
		url,
		port,
		categories
	});

	// Record diagnostic info to stage log for debugging
	scanStageLogger?.recordEvent('lighthouse_start', {
		url,
		port,
		categories
	});

	const flags = {
		port,
		output: 'json' as const,
		onlyCategories: categories,
		// Disable throttling for container environments
		throttling: {
			cpuSlowdownMultiplier: 1,
			rttMs: 0,
			throughputKbps: 0
		},
		// Allow Lighthouse to reset storage between runs when reusing Chrome.
		disableStorageReset: false,
		formFactor: 'desktop' as const,
		screenEmulation: {
			mobile: false,
			width: 1280,
			height: 720,
			deviceScaleFactor: 1,
			disabled: false
		}
	};

	const config = {
		extends: 'lighthouse:default',
		settings: {
			onlyCategories: categories,
			formFactor: 'desktop' as const,
			throttling: {
				cpuSlowdownMultiplier: 1
			},
			screenEmulation: {
				mobile: false,
				width: 1280,
				height: 720,
				deviceScaleFactor: 1,
				disabled: false
			}
		}
	};

	try {
		const runnerResult = await lighthouse(url, flags, config);

		if (!runnerResult) {
			logger.error('Lighthouse returned null/undefined');
			throw new Error('Lighthouse did not return any result');
		}

		if (!runnerResult.lhr) {
			logger.error('Lighthouse result missing lhr', {
				hasReport: !!runnerResult.report,
				resultKeys: Object.keys(runnerResult)
			});
			throw new Error('Lighthouse did not return LHR (Lighthouse Report)');
		}

		const lhr = runnerResult.lhr as unknown as LighthouseResult;

		const auditsCount = Object.keys(lhr.audits ?? {}).length;
		const categoriesCount = Object.keys(lhr.categories ?? {}).length;
		const categoryIds = Object.keys(lhr.categories ?? {});

		logger.info('Lighthouse completed', {
			finalUrl: lhr.finalUrl,
			requestedUrl: lhr.requestedUrl,
			auditsCount,
			categoriesCount,
			categoryIds
		});

		// Record to stage log for debugging
		scanStageLogger?.recordEvent('lighthouse_completed', {
			finalUrl: lhr.finalUrl,
			requestedUrl: lhr.requestedUrl,
			auditsCount,
			categoriesCount,
			categoryIds
		});

		// Log a sample of audit results for debugging
		const audits = lhr.audits ?? {};
		const auditKeys = Object.keys(audits);
		if (auditKeys.length > 0) {
			const sampleAudits = auditKeys.slice(0, 5).map((key) => {
				const audit = audits[key];
				return {
					id: audit?.id,
					score: audit?.score,
					scoreDisplayMode: audit?.scoreDisplayMode
				};
			});
			logger.info('Sample audits', { sampleAudits });
			scanStageLogger?.recordEvent('lighthouse_sample_audits', {
				sampleAudits
			});
		} else {
			logger.warn('No audits returned from Lighthouse');
			scanStageLogger?.recordEvent('lighthouse_no_audits', {});
		}

		return lhr;
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		logger.error('Lighthouse execution failed', {
			error: errorMessage,
			stack: error instanceof Error ? error.stack : undefined
		});
		scanStageLogger?.recordEvent('lighthouse_error', {
			error: errorMessage
		});
		throw error;
	}
}
