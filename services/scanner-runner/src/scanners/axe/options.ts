/**
 * Default milliseconds to wait after load for dynamic content to render.
 * SPAs and JS-heavy sites often need this extra time for React/Vue/Svelte hydration.
 * Reduced from 500ms to 50ms since most modern frameworks hydrate quickly.
 */
export const DEFAULT_DYNAMIC_CONTENT_WAIT_MS = 50;

/**
 * Maximum time to wait for networkidle state before proceeding with scan.
 * Prevents hanging on sites with persistent connections (WebSockets, long-polling).
 */
export const NETWORKIDLE_TIMEOUT_MS = 5_000;

/**
 * Options for the axe scanner.
 * Pass via SCANNER_OPTIONS environment variable as JSON.
 */
export interface AxeOptions {
	/**
	 * Milliseconds to wait after networkidle for dynamic content to render.
	 * Increase for heavy SPAs (React, Vue, Svelte) that need more hydration time.
	 * Set to 0 to skip the wait entirely.
	 * Default: 500
	 */
	dynamicContentWaitMs?: number;

	/**
	 * axe-core rules to disable. Useful for suppressing known false positives.
	 * Example: ["color-contrast", "region"]
	 */
	disabledRules?: string[];

	/**
	 * WCAG tags to limit the scan to. If not set, all rules run.
	 * Example: ["wcag2a", "wcag2aa", "wcag21aa"]
	 */
	runOnlyTags?: string[];
}

export function parseAxeOptions(raw: unknown): AxeOptions {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		return { dynamicContentWaitMs: DEFAULT_DYNAMIC_CONTENT_WAIT_MS };
	}

	const record = raw as Record<string, unknown>;
	const options: AxeOptions = {};

	// Parse dynamicContentWaitMs
	const waitMs = record.dynamicContentWaitMs;
	if (typeof waitMs === 'number' && waitMs >= 0) {
		options.dynamicContentWaitMs = waitMs;
	} else {
		options.dynamicContentWaitMs = DEFAULT_DYNAMIC_CONTENT_WAIT_MS;
	}

	// Parse disabledRules
	const disabledRules = record.disabledRules;
	if (Array.isArray(disabledRules)) {
		options.disabledRules = disabledRules.filter(
			(r): r is string => typeof r === 'string' && r.length > 0
		);
	}

	// Parse runOnlyTags
	const runOnlyTags = record.runOnlyTags;
	if (Array.isArray(runOnlyTags)) {
		options.runOnlyTags = runOnlyTags.filter(
			(t): t is string => typeof t === 'string' && t.length > 0
		);
	}

	return options;
}

export async function withTimeoutFallback<T>(
	operation: Promise<T>,
	timeoutMs: number,
	fallback: () => T
): Promise<T> {
	let timeout: ReturnType<typeof setTimeout> | undefined;

	try {
		return await Promise.race([
			operation,
			new Promise<T>((resolve) => {
				timeout = setTimeout(() => {
					resolve(fallback());
				}, timeoutMs);
			})
		]);
	} finally {
		if (timeout !== undefined) {
			clearTimeout(timeout);
		}
	}
}
