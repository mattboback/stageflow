// Generated from real sample-site scans by just demo-fixture.
export const DEMO_SUMMARY = {
	pages: 4,
	scannersRun: 5,
	totalIssues: 33,
	serious: 14,
	moderate: 8,
	newIssues: 3,
	fixedIssues: 1,
	unchangedIssues: 30,
	baselineScore: 60,
	currentScore: 57,
	scoreDelta: -3,
	terminal:
		'stageflow diff baseline.json current.json --fail-on-new=any\nNew: 3 · Fixed: 1 · Unchanged: 30'
} as const;
