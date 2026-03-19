export function buildExpirationLabel(updatedAt: string | null): string | null {
	if (!updatedAt) return null;

	const timestamp = new Date(updatedAt).getTime();
	if (!Number.isFinite(timestamp)) return null;

	return formatExpirationLabel(new Date(timestamp + 24 * 60 * 60 * 1000));
}

function formatExpirationLabel(date: Date) {
	try {
		return date.toLocaleString(undefined, {
			dateStyle: 'medium',
			timeStyle: 'short'
		});
	} catch {
		return date.toISOString();
	}
}
