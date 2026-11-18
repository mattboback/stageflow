import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import type { UnifiedReport } from './types/unified-report';
import { computeProjectDiff } from './projects';
import { buildVerificationRequest, evaluateVerification } from './verification';

type Scanner = UnifiedReport['scanners'][number];
type Page = NonNullable<Scanner['pages']>[number];
const fixture = JSON.parse(
	fs.readFileSync(path.resolve('../../qa/fixtures/comparison-certainty.json'), 'utf8')
) as {
	baseline: UnifiedReport;
	cases: {
		name: string;
		rules: NonNullable<Page['rules']>;
		currentSubjects: string[];
		omitScanner?: boolean;
		scannerStatus?: Scanner['status'];
		pageStatus?: Page['status'];
		expected: {
			fixed: number;
			unchanged: number;
			new: number;
			limited: boolean;
			verification: string;
		};
	}[];
};

describe('shared comparison certainty fixtures', () => {
	it.each(fixture.cases)('$name', (scenario) => {
		const baseline = structuredClone(fixture.baseline);
		const current = structuredClone(baseline);
		current.meta.jobId = 'current-job';
		current.issues = scenario.currentSubjects.length
			? [
					{
						...baseline.issues[0]!,
						id: 'renamed-issue',
						elementCount: scenario.currentSubjects.length,
						occurrences: scenario.currentSubjects.map((elementId) => ({ elementId }))
					}
				]
			: [];
		const scanner = current.scanners[0]!;
		scanner.status = scenario.scannerStatus ?? 'success';
		scanner.pages![0]!.status = scenario.pageStatus ?? 'success';
		scanner.pages![0]!.rules = scenario.rules;
		if (scenario.omitScanner) current.scanners = [];
		const result = computeProjectDiff('original-job', baseline, 'current-job', current);
		expect(result.delta.fixedIssues).toBe(scenario.expected.fixed);
		expect(result.delta.unchangedIssues).toBe(scenario.expected.unchanged);
		expect(result.delta.newIssues).toBe(scenario.expected.new);
		expect(Boolean(result.notComparedPages?.length || result.skippedScanners?.length)).toBe(
			scenario.expected.limited
		);
		if (scenario.expected.limited) expect(result.delta.scoreDelta).toBeUndefined();
		const reverse = computeProjectDiff('current-job', current, 'original-job', baseline);
		expect(reverse.delta.fixedIssues).toBe(scenario.expected.new);
		expect(reverse.delta.newIssues).toBe(scenario.expected.fixed);
		expect(reverse.delta.unchangedIssues).toBe(scenario.expected.unchanged);
		if (scenario.expected.limited) expect(reverse.delta.scoreDelta).toBeUndefined();
		current.meta.scanContext!.purpose = 'verification';
		const request = buildVerificationRequest(baseline, baseline.issues[0]!, 'current-job');
		expect(evaluateVerification(request, current).status).toBe(scenario.expected.verification);
	});
});
