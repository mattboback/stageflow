import type { ScreenshotArtifact } from '$lib/types/scan';

const PAGE_OVERVIEW_ID = '__page_overview__';

interface IssueScreenshotParams {
	screenshots: ScreenshotArtifact[];
	scannerId: string;
	ruleId: string;
	pageId: string;
}

export function getIssueScreenshotUrl({
	screenshots,
	scannerId,
	ruleId,
	pageId
}: IssueScreenshotParams): string | null {
	const match = screenshots.find(
		(shot) =>
			shot.violation_id === ruleId &&
			shot.page_id === pageId &&
			(!shot.scanner_type || shot.scanner_type === scannerId)
	);
	return match?.url ?? null;
}

export function getPageOverviewUrl(
	screenshots: ScreenshotArtifact[],
	pageId: string,
	preferredScannerOrder: string[] = ['axe']
): string | null {
	const matches = screenshots.filter(
		(shot) => shot.violation_id === PAGE_OVERVIEW_ID && shot.page_id === pageId
	);
	if (matches.length === 0) return null;

	for (const scannerId of preferredScannerOrder) {
		const preferred = matches.find((shot) => shot.scanner_type === scannerId);
		if (preferred) return preferred.url;
	}

	return matches.find((shot) => shot.scanner_type)?.url ?? matches[0].url;
}
