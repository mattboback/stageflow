import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';
import type { UnifiedReport } from '../types/unified-report';
import { buildOccurrenceModeReport } from './occurrence-mode';
import {
	computeSiteScore,
	explainScore,
	penaltyShareByScanner,
	scoreIfFixed,
	sharePercents,
	type ScoreCounts
} from './score';

// Vitest runs with cwd at clients/web. The orchestrator's Go tests read the
// same cases (services/orchestrator/internal/aggregate/score_cases_test.go).
const cases = JSON.parse(
	readFileSync(path.resolve('../../libs/contracts/report/fixtures/score-cases.json'), 'utf8')
) as { cases: { name: string; counts: ScoreCounts; pagesScanned: number; score: number }[] };

const demo = JSON.parse(
	readFileSync(path.resolve('public/demo/report.json'), 'utf8')
) as UnifiedReport;

describe('computeSiteScore', () => {
	it.each(cases.cases)('matches the orchestrator: $name', ({ counts, pagesScanned, score }) => {
		expect(computeSiteScore(counts, pagesScanned)).toBe(score);
	});
});

describe('explainScore', () => {
	it('reproduces the score the demo report shows from its actionable findings', () => {
		const model = explainScore(demo);
		expect(model).toMatchObject({ score: 57, pagesScanned: 4 });
		expect(model?.issues).toHaveLength(demo.assessment?.actionable ?? -1);
	});

	it('refuses a score these findings do not produce', () => {
		// The contract fixture is hand-written: it records 82, its findings give 69.
		expect(explainScore(loadAllScansFixture())).toBeNull();
	});

	it('refuses reports that show no score', () => {
		const verification: UnifiedReport = {
			...demo,
			meta: {
				...demo.meta,
				scanContext: { ...demo.meta.scanContext!, purpose: 'verification' }
			}
		};
		const partial: UnifiedReport = {
			...demo,
			scanners: demo.scanners.map((scanner, index) =>
				index === 0 ? { ...scanner, status: 'failed' } : scanner
			)
		};
		expect(explainScore(verification)).toBeNull();
		expect(explainScore(partial)).toBeNull();
	});

	it('counts the findings an older summary score counted when there is no assessment', () => {
		const legacy: UnifiedReport = {
			...demo,
			summary: { ...demo.summary, score: computeSiteScore({ critical: 3 }, 4) },
			issues: demo.issues
				.filter((issue) => issue.severity === 'critical')
				.concat({ ...demo.issues[0]!, id: 'unconfirmed', scannerData: { axeIncomplete: true } })
		};
		delete legacy.assessment;
		const model = explainScore(legacy);
		expect(model?.score).toBe(computeSiteScore({ critical: 3 }, 4));
		expect(model?.issues.map((issue) => issue.id)).not.toContain('unconfirmed');
	});
});

describe('scoreIfFixed', () => {
	const model = explainScore(demo)!;

	it('scores the remaining findings', () => {
		expect(scoreIfFixed(model, [])).toBe(57);
		expect(
			scoreIfFixed(
				model,
				model.issues.map((issue) => issue.id)
			)
		).toBe(100);
	});

	it('maps an occurrence back to the finding it was split from', () => {
		const critical = model.issues.find((issue) => issue.severity === 'critical')!;
		const source = {
			...critical,
			occurrences: [{ elementId: 'occ-v2-first' }, { elementId: 'occ-v2-second' }]
		};
		const [first, second] = buildOccurrenceModeReport({ ...demo, issues: [source] }).issues;
		expect(second?.id).toBe(`${critical.id}--occ-occ-v2-second`);
		const fixedOne = scoreIfFixed(model, [second!.id]);
		expect(fixedOne).toBeGreaterThan(57);
		expect(fixedOne).toBe(scoreIfFixed(model, [critical.id]));
		expect(scoreIfFixed(model, [first!.id, second!.id])).toBe(fixedOne);
	});
});

describe('penaltyShareByScanner', () => {
	it('splits the demo penalty by scanner, largest first', () => {
		const shares = penaltyShareByScanner(explainScore(demo)!);
		expect(shares.map(({ scanner, scoreWithout }) => [scanner, scoreWithout])).toEqual([
			['axe', 67],
			['security-headers', 66],
			['lighthouse', 59],
			['link-checker', 58],
			['seo', 58]
		]);
		expect(sharePercents(shares.map(({ share }) => share))).toEqual([44, 41, 7, 4, 4]);
	});
});

describe('sharePercents', () => {
	it('rounds shares to whole percents that add up to 100', () => {
		expect(sharePercents([1 / 3, 1 / 3, 1 / 3])).toEqual([34, 33, 33]);
		expect(sharePercents([0.5, 0.5])).toEqual([50, 50]);
		expect(sharePercents([])).toEqual([]);
	});
});
