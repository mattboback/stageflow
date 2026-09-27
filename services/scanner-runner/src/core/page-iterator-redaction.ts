/**
 * Redaction helpers for page iteration: every PageEntry and PageScanResult
 * leaves the iterator with secret values already masked.
 */

import {
	redactDynamicStringValues,
	redactStringValues,
	redactSecretValues
} from '../utils/secret-redaction';
import type { PageEntry, PageScanResult, Provenance, ScannerLogger } from './types';

export function redactPageEntry(
	pageEntry: PageEntry,
	redact: (value: string) => string
): PageEntry {
	const safePageEntry = redactStringValues(pageEntry, redact);
	if (pageEntry.metadata !== undefined) {
		safePageEntry.metadata = redactDynamicStringValues(pageEntry.metadata, redact);
	}
	return safePageEntry;
}

export function redactPageScanResult(
	result: PageScanResult,
	redact: (value: string) => string
): PageScanResult {
	const safeResult = redactStringValues(result, redact);

	for (const [index, issue] of result.issues.entries()) {
		const safeIssue = safeResult.issues[index];
		if (safeIssue && issue.metadata !== undefined) {
			safeIssue.metadata = redactDynamicStringValues(issue.metadata, redact);
		}
	}

	if (result.rawResults !== undefined) {
		safeResult.rawResults = redactDynamicStringValues(result.rawResults, redact);
	}

	return safeResult;
}

/** Collect all values before any navigation, including actions on later pages. */
export function createLiteralRedactor(provenance: Provenance): (value: string) => string {
	const values = new Set<string>();
	const actions = [
		...(provenance.auth?.mode === 'form' ? provenance.auth.steps : []),
		...provenance.pages.flatMap((page) => page.pre_scan_actions ?? [])
	];
	for (const action of actions) {
		if ((action.type === 'fill' || action.type === 'select') && typeof action.value === 'string') {
			values.add(action.value);
		}
	}
	return (value) => redactSecretValues(value, values);
}

/** Wrap a page's logger without mutating shared scanner state during parallel scans. */
export function createRedactedLogger(
	logger: ScannerLogger,
	redact: (value: string) => string
): ScannerLogger {
	const log =
		(level: keyof ScannerLogger) =>
		(message: string, meta?: Record<string, unknown>): void => {
			logger[level](
				redact(message),
				meta === undefined ? undefined : redactDynamicStringValues(meta, redact)
			);
		};
	return { debug: log('debug'), info: log('info'), warn: log('warn'), error: log('error') };
}
