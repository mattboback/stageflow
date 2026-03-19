import type { ScanResult } from "$lib/types/scan";
import type { UnifiedReport } from "$lib/types/unified-report";

import ScannersView from "$lib/components/report/ScannersView.svelte";
import { render } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

const report: UnifiedReport = {
	version: "2.0.0",
	meta: { jobId: "job-1" },
	summary: {
		totalIssues: 1,
		bySeverity: { critical: 0, serious: 0, moderate: 1, minor: 0, info: 0 },
		pagesScanned: 1,
		pagesWithIssues: 1,
		lighthouseCategories: [
			{ id: "performance", title: "Performance", avgScore: 0.8 },
		],
	},
	scanners: [{ id: "lighthouse", status: "success", issueCount: 1 }],
	pages: [
		{
			id: "page-1",
			url: "http://example.com",
			issueCount: 1,
			durationMs: 1000,
		},
	],
	issues: [
		{
			id: "issue-1",
			scanner: "lighthouse",
			ruleId: "speed-index",
			severity: "moderate",
			title: "Slow speed index",
			description: "Speed index is slow",
			pageId: "page-1",
			pageUrl: "http://example.com",
			elementCount: 1,
		},
	],
};

const job: ScanResult = {
	id: "job-1",
	state: "DONE",
	created_at: new Date().toISOString(),
	updated_at: new Date().toISOString(),
	artifacts: {
		report_json: "http://example.com/report.json",
		report_html: "http://example.com/report.html",
		scanner_artifacts: {
			lighthouse: {
				scanner_type: "lighthouse",
				results_json: "http://example.com/lh.json",
			},
		},
	},
};

describe("ScannersView", () => {
	it("renders lighthouse summary when selected", () => {
		const { getByText } = render(ScannersView, {
			props: {
				report,
				job,
				activeScanner: "lighthouse",
				onSelectScanner: () => undefined,
			},
		});

		expect(getByText("Lighthouse categories")).toBeInTheDocument();
		expect(getByText("Performance")).toBeInTheDocument();
	});
});
