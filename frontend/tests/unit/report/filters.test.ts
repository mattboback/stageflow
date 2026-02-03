import type { IssueDetail } from '$lib/types/unified-report';

import { filterIssues, groupIssues, sortIssues } from '$lib/report';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const fixturePath = path.resolve(
	__dirname,
	'../../../../../packages/contracts/report/fixtures/unified-report.v2.all-scans.json'
);
const fixture = JSON.parse(readFileSync(fixturePath, 'utf8')) as { issues: IssueDetail[] };

describe('report filters', () => {
	it('filters by scanner and severity', () => {
		const filtered = filterIssues(fixture.issues, {
			scannerId: 'axe',
			severity: 'critical'
		});
		expect(filtered.length).toBeGreaterThan(0);
		expect(filtered.every((issue) => issue.scanner === 'axe')).toBe(true);
		expect(filtered.every((issue) => issue.severity === 'critical')).toBe(true);
	});

	it('filters by category and query', () => {
		const filtered = filterIssues(fixture.issues, {
			category: 'security',
			query: 'header'
		});
		expect(filtered.length).toBeGreaterThan(0);
		expect(filtered.every((issue) => issue.category === 'security')).toBe(true);
	});
});

describe('report grouping + sorting', () => {
	it('groups issues by rule', () => {
		const grouped = groupIssues(fixture.issues, 'rule');
		expect(grouped.length).toBeGreaterThan(0);
		expect(grouped[0]?.issues.length).toBeGreaterThan(0);
	});

	it('sorts issues by severity', () => {
		const sorted = sortIssues(fixture.issues, 'severity');
		expect(sorted[0]).toBeDefined();
	});
});
