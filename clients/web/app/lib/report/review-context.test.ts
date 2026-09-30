import { describe, expect, it, vi } from 'vitest';
import { webcrypto } from 'node:crypto';
import { createReviewContext, findPriorReview, type ReviewSubject } from './review-context';
import { loadAllScansFixture } from '../../test/load-fixture';

const subject: ReviewSubject = {
	subjectKey: 'axe|rule|page-v2-first|occ-v2-first',
	evidenceSignature: 'review-v1:evidence',
	executionSignature: 'review-v1:settings',
	scannerVersion: '1'
};

describe('review carry-forward', () => {
	it('suggests matching evidence without creating a current decision', () => {
		const records = {
			old: { first: { ...subject, verdict: 'pass' as const, at: '2026-09-30T00:00:00Z' } }
		};
		expect(findPriorReview(records, 'new', 'renamed-display-id', subject)).toMatchObject({
			jobId: 'old',
			stale: false
		});
		expect(records).not.toHaveProperty('new');
	});
	it('marks changed evidence, execution and scanner versions stale', () => {
		const records = {
			old: { first: { ...subject, verdict: 'pass' as const, at: '2026-09-30T00:00:00Z' } }
		};
		for (const change of [
			{ evidenceSignature: 'changed' },
			{ executionSignature: 'changed' },
			{ scannerVersion: '2' },
			{ evidenceSignature: undefined }
		]) {
			expect(
				findPriorReview(records, 'new', 'first', { ...subject, ...change } as ReviewSubject)?.stale
			).toBe(true);
		}
	});
	it('keeps legacy records as historical, never confirmation eligible', () => {
		expect(
			findPriorReview(
				{ old: { first: { verdict: 'fail', at: '2026-09-30' } } },
				'new',
				'first',
				subject
			)?.stale
		).toBe(true);
	});
	it('requires screenshot evidence for screenshot-derived measurements', () => {
		const record = {
			...subject,
			verdict: 'pass' as const,
			at: '2026-09-30',
			screenshotDigest: 'bytes',
			measurement: { fg: '#000000', bg: '#ffffff', ratio: 21 }
		};
		expect(findPriorReview({ old: { first: record } }, 'new', 'first', subject)?.stale).toBe(true);
		expect(
			findPriorReview({ old: { first: record } }, 'new', 'first', {
				...subject,
				screenshotDigest: 'bytes'
			})?.stale
		).toBe(false);
	});
	it('legacy reports cannot invent execution or evidence signatures', async () => {
		const context = await createReviewContext(loadAllScansFixture());
		for (const item of Object.values(context.subjects))
			expect(item.executionSignature).toBeUndefined();
	});
});

it('hashes screenshot and execution evidence and preserves stable occurrence subjects', async () => {
	vi.stubGlobal('crypto', webcrypto);
	try {
		const report = structuredClone(loadAllScansFixture());
		const issue = report.issues[0];
		if (!issue) throw new Error('Fixture requires an issue');
		issue.pageId = 'page-v2-test';
		issue.occurrences = [
			{ elementId: 'occ-v2-test', selector: '#target', html: '<span id="target">Text</span>' }
		];
		report.meta.scanContext = {
			expectedPages: ['page-v2-test'],
			identityScope: 'path',
			scannerConfigFingerprints: { axe: 'options' },
			purpose: 'full',
			authenticated: false,
			inputType: 'urls'
		};
		report.scanners = report.scanners.map((scanner) => ({
			...scanner,
			toolVersion: 'same-version'
		}));
		const original = await createReviewContext(report, 'project', 'config', {
			'page-v2-test': 'image-one'
		});
		const changedImage = await createReviewContext(report, 'project', 'config', {
			'page-v2-test': 'image-two'
		});
		expect(original.subjects[issue.id]?.subjectKey).toContain('occ-v2-test');
		expect(original.subjects[issue.id]?.evidenceSignature).toMatch(/^review-v1:[a-f0-9]{64}$/);
		expect(original.subjects[issue.id]?.evidenceSignature).not.toBe(
			changedImage.subjects[issue.id]?.evidenceSignature
		);
		report.meta.profile = 'staging';
		const changedProfile = await createReviewContext(report, 'project', 'config', {
			'page-v2-test': 'image-one'
		});
		expect(original.subjects[issue.id]?.executionSignature).not.toBe(
			changedProfile.subjects[issue.id]?.executionSignature
		);
	} finally {
		vi.unstubAllGlobals();
	}
});
