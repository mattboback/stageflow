/**
 * Redaction helpers for page iteration: every PageEntry and PageScanResult
 * leaves the iterator with secret values already masked.
 */

import { redactDynamicStringValues, redactStringValues } from '../utils/secret-redaction';
import type { createSecretsResolver } from './secrets-resolver';
import type { PageEntry, PageScanResult } from './types';

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

export function registerPageLiteralValues(
	pageEntry: PageEntry,
	secretsResolver: ReturnType<typeof createSecretsResolver>
): void {
	for (const action of pageEntry.pre_scan_actions ?? []) {
		if ((action.type === 'fill' || action.type === 'select') && typeof action.value === 'string') {
			secretsResolver.resolveValue(action.value);
		}
	}
}
