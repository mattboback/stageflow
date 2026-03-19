import type { ScreenshotArtifact } from "$lib/types/scan";

import {
	getIssueScreenshotUrl,
	getPageOverviewUrl,
} from "$lib/report/screenshots";
import { describe, expect, it } from "vitest";

describe("report screenshots", () => {
	it("resolves issue screenshots by issue id", () => {
		const screenshots: ScreenshotArtifact[] = [
			{
				kind: "violation",
				issue_id: "issue-a",
				occurrence_index: 0,
				artifact_id: "ss-issue-a",
				scanner_id: "axe",
				page_id: "page-1",
				url: "https://example.com/a.webp",
			},
			{
				kind: "violation",
				issue_id: "issue-b",
				occurrence_index: 1,
				artifact_id: "ss-issue-b",
				scanner_id: "axe",
				page_id: "page-1",
				url: "https://example.com/b.webp",
			},
		];

		expect(
			getIssueScreenshotUrl({
				screenshots,
				scannerId: "axe",
				issueId: "issue-b",
				pageId: "page-1",
			}),
		).toBe("https://example.com/b.webp");
	});

	it("does not return screenshot when scanner id differs", () => {
		const screenshots: ScreenshotArtifact[] = [
			{
				kind: "violation",
				issue_id: "issue-a",
				occurrence_index: 0,
				artifact_id: "ss-issue-a",
				scanner_id: "lighthouse",
				page_id: "page-1",
				url: "https://example.com/a.webp",
			},
		];

		expect(
			getIssueScreenshotUrl({
				screenshots,
				scannerId: "axe",
				issueId: "issue-a",
				pageId: "page-1",
			}),
		).toBeNull();
	});

	it("selects preferred page overview scanner when multiple exist", () => {
		const screenshots: ScreenshotArtifact[] = [
			{
				kind: "page_overview",
				artifact_id: "page-overview:lighthouse:page-1",
				scanner_id: "lighthouse",
				page_id: "page-1",
				url: "https://example.com/lh.webp",
			},
			{
				kind: "page_overview",
				artifact_id: "page-overview:axe:page-1",
				scanner_id: "axe",
				page_id: "page-1",
				url: "https://example.com/axe.webp",
			},
		];

		expect(
			getPageOverviewUrl(screenshots, "page-1", ["axe", "lighthouse"]),
		).toBe("https://example.com/axe.webp");
	});
});
