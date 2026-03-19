import type { IssueDetail, PageSummary } from "$lib/types/unified-report";

import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { summarizeIssuesByPage, summarizeIssuesByRule } from "$lib/report";
import { describe, expect, it } from "vitest";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const fixturePath = path.resolve(
	__dirname,
	"../../../../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json",
);
const fixture = JSON.parse(readFileSync(fixturePath, "utf8")) as {
	issues: IssueDetail[];
	pages: PageSummary[];
};

describe("scanner summaries", () => {
	it("summarizes issues by rule", () => {
		const summary = summarizeIssuesByRule(fixture.issues);
		expect(summary.length).toBeGreaterThan(0);
		expect(summary[0]?.count).toBeGreaterThan(0);
	});

	it("summarizes issues by page", () => {
		const pagesById = Object.fromEntries(
			fixture.pages.map((page) => [page.id, page]),
		);
		const summary = summarizeIssuesByPage(fixture.issues, pagesById);
		expect(summary.length).toBeGreaterThan(0);
		expect(summary[0]?.label.length).toBeGreaterThan(0);
	});
});
