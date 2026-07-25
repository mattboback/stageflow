import type { ScreenshotArtifact } from '../types/scan';
import type { PageOverviewElement, PageSummary } from '../types/unified-report';

interface IssueScreenshotParams {
	screenshots: ScreenshotArtifact[];
	scannerId: string;
	issueId: string;
	pageId: string;
}

export function getIssueScreenshotUrl({
	screenshots,
	scannerId,
	issueId,
	pageId
}: IssueScreenshotParams): string | null {
	const exactMatch = screenshots.find(
		(shot) =>
			shot.kind === 'violation' &&
			shot.issue_id === issueId &&
			shot.page_id === pageId &&
			shot.scanner_id === scannerId
	);
	return exactMatch?.url ?? null;
}

export function getPageOverviewUrl(
	screenshots: ScreenshotArtifact[],
	pageId: string,
	preferredScannerOrder: string[] = ['axe']
): string | null {
	const matches = screenshots.filter(
		(shot) => shot.kind === 'page_overview' && shot.page_id === pageId
	);
	if (matches.length === 0) return null;

	for (const scannerId of preferredScannerOrder) {
		const preferred = matches.find((shot) => shot.scanner_id === scannerId);
		if (preferred) return preferred.url;
	}

	return matches[0]?.url ?? null;
}

/*
 * For the primary occurrence (nodeIndex 0) any box for the issue is better
 * than nothing; for later occurrences only an exact nodeIndex match is safe,
 * otherwise every card would show the first element again.
 */
export function findOverviewElement(
	page: PageSummary | null,
	issueId: string,
	nodeIndex = 0
): PageOverviewElement | null {
	const elements = page?.pageOverview?.elements ?? [];
	const exact = elements.find((el) => el.issueId === issueId && el.nodeIndex === nodeIndex);
	if (exact) return exact;
	if (nodeIndex === 0) {
		return elements.find((el) => el.issueId === issueId) ?? null;
	}
	return null;
}
