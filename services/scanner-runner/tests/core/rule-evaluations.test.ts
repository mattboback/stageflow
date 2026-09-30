import { afterEach, describe, expect, it, vi } from 'vitest';
import { axeRuleEvaluations, boundRuleEvaluations } from '../../src/core/rule-evaluations';
import { occurrenceFingerprint } from '../../src/core/finding-identity';
import { lighthouseRuleEvaluations } from '../../src/scanners/lighthouse/rule-evaluations';
import { extractAuditNodes } from '../../src/scanners/lighthouse/result-parser';
import type { LighthouseAudit } from '../../src/scanners/lighthouse/types';

afterEach(() => vi.unstubAllEnvs());

describe('compact evaluated rule coverage', () => {
	it('retains all four axe outcome classes with the report occurrence identity', () => {
		vi.stubEnv('SCAN_PURPOSE', 'verification');
		const result = axeRuleEvaluations(
			{
				violations: [{ id: 'image-alt', nodes: [{ target: ['#logo'] }] }],
				passes: [{ id: 'image-alt', nodes: [{ target: ['#fixed'] }] }],
				incomplete: [{ id: 'color-contrast', nodes: [{ target: ['#text'] }] }],
				inapplicable: [{ id: 'video-caption' }]
			},
			'page-v2-home'
		);
		expect(result.map((rule) => rule.outcome)).toEqual([
			'fail',
			'incomplete',
			'pass',
			'inapplicable'
		]);
		expect(result[0]?.subjectIds).toEqual([
			occurrenceFingerprint('axe', 'image-alt', 'page-v2-home', ['#logo'])
		]);
		expect(result[2]?.subjectIds).toEqual([
			occurrenceFingerprint('axe', 'image-alt', 'page-v2-home', ['#fixed'])
		]);
		expect(JSON.stringify(result)).not.toContain('nodes');
	});

	it('bounds rule count, subjects per rule and total subject identities', () => {
		const result = boundRuleEvaluations(
			Array.from({ length: 600 }, (_, rule) => ({
				ruleId: `rule-${rule}`,
				outcome: 'fail' as const,
				subjectIds: Array.from({ length: 100 }, (_, subject) => `${rule}:${subject}`)
			}))
		);
		expect(result).toHaveLength(512);
		expect(result.every((rule) => (rule.subjectIds?.length ?? 0) <= 32)).toBe(true);
		expect(result.reduce((count, rule) => count + (rule.subjectIds?.length ?? 0), 0)).toBe(1024);
	});

	it('records Lighthouse manual, error and informative audits without calling them passes', () => {
		const audit = (
			id: string,
			scoreDisplayMode: string,
			score: number | null
		): LighthouseAudit => ({ id, title: id, description: '', scoreDisplayMode, score });
		const rules = lighthouseRuleEvaluations(
			{
				requestedUrl: 'https://example.com',
				finalUrl: 'https://example.com',
				fetchTime: '',
				audits: {
					pass: audit('pass', 'binary', 1),
					fail: audit('fail', 'binary', 0),
					manual: audit('manual', 'manual', null),
					error: audit('error', 'error', null),
					informational: audit('informational', 'informative', null),
					irrelevant: audit('irrelevant', 'notApplicable', null)
				}
			},
			'page'
		);
		expect(Object.fromEntries(rules.map((rule) => [rule.ruleId, rule.outcome]))).toEqual({
			pass: 'pass',
			fail: 'fail',
			manual: 'incomplete',
			error: 'error',
			informational: 'incomplete',
			irrelevant: 'inapplicable'
		});
	});

	it('anchors robots evidence to the observed meta tag and omits an ambiguous charset crop', () => {
		const audit: LighthouseAudit = {
			id: 'is-crawlable',
			title: 'Indexing',
			description: '',
			score: 0,
			scoreDisplayMode: 'binary',
			details: {
				type: 'table',
				items: [
					{ node: { selector: 'head > meta', snippet: '<meta name="robots" content="noindex">' } }
				]
			}
		};
		expect(extractAuditNodes(audit)[0]?.selector).toBe('meta[name="robots" i]');
		audit.details = {
			type: 'table',
			items: [{ node: { selector: 'head > meta', snippet: '<meta charset="utf-8">' } }]
		};
		expect(extractAuditNodes(audit)).toEqual([]);
	});
});
