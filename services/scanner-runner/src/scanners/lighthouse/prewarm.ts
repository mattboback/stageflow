export function isLighthousePrewarmEnabled(): boolean {
	const raw = process.env.LIGHTHOUSE_PREWARM?.trim().toLowerCase();
	return raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on';
}
