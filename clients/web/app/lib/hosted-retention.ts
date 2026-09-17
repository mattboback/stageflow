const HOSTED_ARTIFACT_RETENTION_MS = 24 * 60 * 60 * 1000;

export function hostedExpiryAt(completedAt: string | undefined): Date | null {
	if (!completedAt) return null;
	const completed = new Date(completedAt);
	if (Number.isNaN(completed.getTime())) return null;
	return new Date(completed.getTime() + HOSTED_ARTIFACT_RETENTION_MS);
}

export function formatHostedExpiry(completedAt: string | undefined): string | null {
	const expiry = hostedExpiryAt(completedAt);
	if (!expiry) return null;
	return expiry.toLocaleString(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}

export function hostedEvidenceExpired(completedAt: string | undefined, now = Date.now()): boolean {
	const expiry = hostedExpiryAt(completedAt);
	return expiry !== null && expiry.getTime() <= now;
}
