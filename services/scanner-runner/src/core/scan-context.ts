import { createHash } from 'node:crypto';
import type { ScanContext as ReportScanContext } from '@stageflow/contracts-report';
import type { Provenance, ScannerConfig } from './types';
import { hasMultipleOrigins } from './finding-identity';

function stable(value: unknown): unknown {
	if (Array.isArray(value)) {
		return value.map(stable);
	}
	if (!value || typeof value !== 'object') {
		return value;
	}
	return Object.fromEntries(
		Object.entries(value)
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([key, item]) => [key, stable(item)])
	);
}

export function scanIdentityIncludesOrigin(urls: string[]): boolean {
	return hasMultipleOrigins(urls) || process.env.SCAN_IDENTITY_SCOPE === 'origin';
}

export function createScanContext(
	config: ScannerConfig,
	provenance: Provenance
): ReportScanContext {
	const safeOptions = safeConfiguration(config.options ?? {});
	const settings = {
		options: safeOptions,
		browser: config.browser,
		wait: provenance.default_wait_for,
		pages: provenance.pages.map((page) => ({
			id: page.id,
			viewport: page.viewport,
			wait: page.wait_for,
			actions: page.pre_scan_actions
		}))
	};
	// Page selection is represented separately: fingerprint the effective shared
	// execution settings, not the number/order of selected pages.
	const sharedSettings = { ...settings, pages: undefined };
	const fingerprint = createHash('sha256')
		.update(JSON.stringify(stable(sharedSettings)))
		.digest('hex');
	return {
		expectedPages: provenance.pages.map((page) => page.id).sort(),
		identityScope: scanIdentityIncludesOrigin(provenance.pages.map((page) => page.url))
			? 'origin'
			: 'path',
		scannerConfigFingerprints: { [config.scannerName]: fingerprint },
		scannerConfigurations: { [config.scannerName]: safeOptions },
		verificationEligiblePages: Object.fromEntries(
			provenance.pages.map((page) => {
				const viewport =
					page.viewport ?? provenance.default_viewport ?? config.browser.defaultViewport;
				const wait = page.wait_for ?? provenance.default_wait_for;
				return [
					page.id,
					viewport.width === 1280 &&
						viewport.height === 720 &&
						!page.pre_scan_actions?.length &&
						(provenance.mode === 'static' ? wait === undefined : wait?.type === 'domcontentloaded')
				];
			})
		),
		pageConfigFingerprints: Object.fromEntries(
			settings.pages.map((page) => [
				page.id,
				createHash('sha256')
					.update(
						JSON.stringify(
							stable({
								viewport:
									page.viewport ?? provenance.default_viewport ?? config.browser.defaultViewport,
								wait: page.wait ?? provenance.default_wait_for,
								actions: page.actions
							})
						)
					)
					.digest('hex')
			])
		),
		viewport: provenance.default_viewport ?? config.browser.defaultViewport,
		purpose: process.env.SCAN_PURPOSE === 'verification' ? 'verification' : 'full',
		authenticated: Boolean(provenance.auth),
		inputType: provenance.mode === 'static' ? 'zip' : 'urls'
	};
}

function safeConfiguration(value: Record<string, unknown>): Record<string, unknown> {
	const safe = (item: unknown): unknown =>
		Array.isArray(item)
			? item.map(safe)
			: item && typeof item === 'object'
				? safeConfiguration(item as Record<string, unknown>)
				: item;
	return Object.fromEntries(
		Object.entries(value)
			.filter(
				([key]) =>
					![
						'auth',
						'authorization',
						'password',
						'secret',
						'token',
						'username',
						'storagestate',
						'apikey',
						'inputvalues'
					].includes(key.toLowerCase().replace(/[^a-z0-9]/g, ''))
			)
			.map(([key, item]) => [key, safe(item)])
	);
}
