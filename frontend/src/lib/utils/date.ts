/**
 * Formats an ISO timestamp string to a human-readable localized date/time.
 * Returns null if the value is falsy or invalid.
 */
export function formatTimestamp(value?: string | null): string | null {
	if (!value) return null;
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	try {
		return date.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
	} catch {
		return date.toISOString();
	}
}

/**
 * Formats a duration in milliseconds to a human-readable string (e.g., "2.5s").
 */
export function formatDuration(durationMs?: number | null): string | null {
	if (!durationMs) return null;
	return `${(durationMs / 1000).toFixed(1)}s`;
}
