/*
 * Headline figures from the committed /demo report.
 *
 * Duplicated here rather than read from public/demo/report.json because the
 * home page would otherwise pull 90 KB of JSON into its bundle to print four
 * numbers. app/lib/demo/demo-fixture.test.ts asserts every field against the
 * real fixture, so a regeneration that changes the report fails CI instead of
 * quietly leaving the marketing page describing an older scan.
 *
 * The previous hero card said "94 → 88 ▼6" for example.com and none of it had
 * ever happened. Fabricated numbers in a hero are the first thing a reviewer
 * discounts, and everything below is checked.
 */
export interface DemoSummary {
	pages: number;
	scannersRun: number;
	totalIssues: number;
	serious: number;
	moderate: number;
}

export const DEMO_SUMMARY: DemoSummary = {
	pages: 3,
	scannersRun: 7,
	totalIssues: 47,
	serious: 3,
	moderate: 6
};
