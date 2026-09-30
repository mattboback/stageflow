import { afterEach, describe, expect, it, vi } from 'vitest';
import { createScanContext } from '../../src/core/scan-context';
import type { Provenance, ScannerConfig } from '../../src/core/types';

function fixtures(): { config: ScannerConfig; provenance: Provenance } {
	return {
		config: {
			jobId: 'job',
			provenancePath: '',
			resultsDir: '',
			scannerName: 'axe',
			concurrency: 1,
			maxRetries: 1,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 2,
				defaultTimeout: 30000,
				pageLoadTimeout: 15000
			},
			storage: { endpoint: '', accessKey: '', secretKey: '', useSSL: false, bucket: '' },
			messaging: { url: '', subjects: { pageCompleted: '', scanCompleted: '', scanFailed: '' } },
			options: {}
		},
		provenance: {
			version: '1',
			job_id: 'job',
			base_url: 'https://example.com',
			mode: 'live',
			default_wait_for: { type: 'domcontentloaded' },
			pages: [{ id: 'page-v2-home', path: '/', url: 'https://example.com/' }]
		}
	};
}

afterEach(() => vi.unstubAllEnvs());

describe('recorded execution context', () => {
	it('allows default URL and ZIP page execution to be replayed', () => {
		const { config, provenance } = fixtures();
		expect(createScanContext(config, provenance).verificationEligiblePages?.['page-v2-home']).toBe(
			true
		);
		provenance.mode = 'static';
		delete provenance.default_wait_for;
		expect(createScanContext(config, provenance).verificationEligiblePages?.['page-v2-home']).toBe(
			true
		);
	});

	it.each(['viewport', 'wait', 'actions'] as const)(
		'marks custom %s execution unavailable for default verification',
		(setting) => {
			const { config, provenance } = fixtures();
			if (setting === 'viewport') {
				provenance.pages[0]!.viewport = { width: 400, height: 800 };
			}
			if (setting === 'wait') {
				provenance.pages[0]!.wait_for = { type: 'selector', selector: '.ready' };
			}
			if (setting === 'actions') {
				provenance.pages[0]!.pre_scan_actions = [{ type: 'click', selector: '#open' }];
			}
			expect(
				createScanContext(config, provenance).verificationEligiblePages?.['page-v2-home']
			).toBe(false);
		}
	);

	it('preserves shared scanner fingerprints across page subsets and separately records page settings', () => {
		const { config, provenance } = fixtures();
		provenance.pages.push({
			id: 'page-v2-about',
			path: '/about',
			url: 'https://example.com/about'
		});
		const full = createScanContext(config, provenance);
		provenance.pages = provenance.pages.slice(0, 1);
		const subset = createScanContext(config, provenance);
		expect(subset.scannerConfigFingerprints).toEqual(full.scannerConfigFingerprints);
		expect(subset.pageConfigFingerprints?.['page-v2-home']).toBe(
			full.pageConfigFingerprints?.['page-v2-home']
		);
		provenance.pages[0]!.viewport = { width: 400, height: 800 };
		expect(createScanContext(config, provenance).pageConfigFingerprints?.['page-v2-home']).not.toBe(
			subset.pageConfigFingerprints?.['page-v2-home']
		);
	});

	it('omits nested authentication values from public replay options', () => {
		const { config, provenance } = fixtures();
		config.options = {
			rule: 'test',
			Authorization: 'secret-value',
			rows: [{ password: 'secret-value', threshold: 1 }],
			storage_state: { cookies: ['secret-value'] }
		};
		const context = createScanContext(config, provenance);
		expect(context.scannerConfigurations?.axe).toEqual({ rule: 'test', rows: [{ threshold: 1 }] });
		expect(JSON.stringify(context)).not.toContain('secret-value');
	});
});
