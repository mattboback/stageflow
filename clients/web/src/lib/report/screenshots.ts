import type { ScreenshotArtifact } from "$lib/types/scan";

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
	pageId,
}: IssueScreenshotParams): string | null {
	const exactMatch = screenshots.find(
		(shot) =>
			shot.kind === "violation" &&
			shot.issue_id === issueId &&
			shot.page_id === pageId &&
			shot.scanner_id === scannerId,
	);
	return exactMatch?.url ?? null;
}

export function getPageOverviewUrl(
	screenshots: ScreenshotArtifact[],
	pageId: string,
	preferredScannerOrder: string[] = ["axe"],
): string | null {
	const matches = screenshots.filter(
		(shot) => shot.kind === "page_overview" && shot.page_id === pageId,
	);
	if (matches.length === 0) return null;

	for (const scannerId of preferredScannerOrder) {
		const preferred = matches.find((shot) => shot.scanner_id === scannerId);
		if (preferred) return preferred.url;
	}

	return matches[0].url;
}
