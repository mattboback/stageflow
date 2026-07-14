import { describe, expect, it } from 'vitest';

import { normalizeReviewVerdicts } from './review-verdict';

describe('normalizeReviewVerdicts', () => {
	it('migrates the legacy top-level contrast measurement fields', () => {
		expect(
			normalizeReviewVerdicts({
				'job-1': {
					'issue-1': {
						verdict: 'pass',
						fg: '#111111',
						bg: '#ffffff',
						ratio: 18.88,
						at: '2026-07-14T10:00:00.000Z'
					}
				}
			})
		).toEqual({
			'job-1': {
				'issue-1': {
					verdict: 'pass',
					at: '2026-07-14T10:00:00.000Z',
					measurement: { fg: '#111111', bg: '#ffffff', ratio: 18.88 }
				}
			}
		});
	});

	it('keeps generic review decisions without a measurement', () => {
		expect(
			normalizeReviewVerdicts({
				'job-1': {
					'manual-1': { verdict: 'fail', at: '2026-07-14T10:00:00.000Z' }
				}
			})
		).toEqual({
			'job-1': {
				'manual-1': { verdict: 'fail', at: '2026-07-14T10:00:00.000Z' }
			}
		});
	});

	it('drops malformed jobs and verdicts', () => {
		expect(
			normalizeReviewVerdicts({
				'job-1': {
					badDecision: { verdict: 'maybe', at: 'today' },
					badTimestamp: { verdict: 'pass', at: '' }
				},
				'job-2': 'not-an-object'
			})
		).toEqual({});
	});
});
