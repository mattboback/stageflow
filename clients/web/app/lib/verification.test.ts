import { describe, expect, it } from 'vitest';
import { loadAllScansFixture } from '../test/load-fixture';
import {
	buildVerificationRequest,
	evaluateVerification,
	loadVerificationRequest,
	parseVerificationRequest,
	saveVerificationRequest,
	verificationUnavailableReason,
	VERIFICATION_STORAGE_KEY
} from './verification';
import type { IssueDetail, UnifiedReport } from './types/unified-report';

const pageId = 'page-v2-home';
const issue: IssueDetail = {
	id: 'original',
	scanner: 'axe',
	ruleId: 'image-alt',
	title: 'Missing alt',
	description: 'Add alternative text',
	severity: 'serious',
	pageId,
	pageUrl: 'https://example.com/',
	elementCount: 1,
	occurrences: [{ elementId: 'occ-v2-original', selector: 'img', html: '<img>' }]
};

function original(): UnifiedReport {
	const fixture = loadAllScansFixture();
	return {
		...fixture,
		meta: {
			...fixture.meta,
			jobId: 'original-job',
			scanContext: {
				expectedPages: [pageId],
				identityScope: 'path',
				scannerConfigFingerprints: { axe: 'settings' },
				pageConfigFingerprints: { [pageId]: 'page-settings' },
				scannerConfigurations: { axe: {} },
				verificationEligiblePages: { [pageId]: true },
				viewport: { width: 1280, height: 720 },
				purpose: 'full',
				authenticated: false,
				inputType: 'urls'
			}
		},
		pages: [{ ...fixture.pages[0]!, id: pageId, url: issue.pageUrl }],
		issues: [issue],
		scanners: [
			{
				id: 'axe',
				name: 'Axe',
				status: 'success',
				toolVersion: '1.0',
				pages: [
					{
						pageId,
						status: 'success',
						finalUrl: issue.pageUrl,
						rules: [{ ruleId: issue.ruleId, outcome: 'fail', subjectIds: ['occ-v2-original'] }]
					}
				]
			}
		]
	};
}

function current(
	outcome: 'pass' | 'fail' | 'incomplete' | 'inapplicable' | 'error' = 'pass'
): UnifiedReport {
	const source = original();
	return {
		...source,
		meta: {
			...source.meta,
			jobId: 'verification-job',
			scanContext: { ...source.meta.scanContext!, purpose: 'verification' }
		},
		issues: [],
		scanners: [
			{
				...source.scanners[0]!,
				pages: [
					{
						pageId,
						status: 'success',
						finalUrl: issue.pageUrl,
						rules: [
							{
								ruleId: issue.ruleId,
								outcome,
								...(outcome === 'fail' || outcome === 'incomplete'
									? { subjectIds: ['occ-v2-original'] }
									: {})
							}
						]
					}
				]
			}
		]
	};
}

describe('targeted verification', () => {
	it.each([
		['pass', 'fixed'],
		['fail', 'still-present'],
		['incomplete', 'needs-review'],
		['inapplicable', 'fixed'],
		['error', 'unverified']
	] as const)('interprets a completed %s evaluation as %s', (outcome, status) => {
		const request = buildVerificationRequest(original(), issue, 'verification-job');
		expect(evaluateVerification(request, current(outcome)).status).toBe(status);
	});

	it.each([
		'missing-page',
		'failed-page',
		'missing-rule',
		'disabled-rule',
		'wrong-page',
		'version',
		'settings',
		'identity',
		'viewport',
		'authentication',
		'page-settings'
	] as const)('does not claim fixed with incompatible %s coverage', (scenario) => {
		const report = current();
		const scanner = report.scanners[0]!;
		const page = scanner.pages![0]!;
		const context = report.meta.scanContext!;
		if (scenario === 'missing-page') scanner.pages = [];
		if (scenario === 'failed-page') page.status = 'failed';
		if (scenario === 'missing-rule') delete page.rules;
		if (scenario === 'disabled-rule') page.rules = [{ ruleId: 'other-rule', outcome: 'pass' }];
		if (scenario === 'wrong-page') page.finalUrl = 'https://example.com/login';
		if (scenario === 'version') scanner.toolVersion = '2.0';
		if (scenario === 'settings') context.scannerConfigFingerprints.axe = 'changed';
		if (scenario === 'identity') context.identityScope = 'origin';
		if (scenario === 'viewport') context.viewport = { width: 400, height: 800 };
		if (scenario === 'authentication') context.authenticated = true;
		if (scenario === 'page-settings') context.pageConfigFingerprints = { [pageId]: 'changed' };
		expect(
			evaluateVerification(buildVerificationRequest(original(), issue, 'verification-job'), report)
				.status
		).toBe('unverified');
	});

	it('requires confirmed subject coverage when the same rule fails elsewhere', () => {
		const report = current('fail');
		report.scanners[0]!.pages![0]!.rules![0]!.subjectIds = ['occ-v2-different'];
		const request = buildVerificationRequest(original(), issue, 'verification-job');
		expect(evaluateVerification(request, report).status).toBe('unverified');
		report.scanners[0]!.pages![0]!.rules!.push({
			ruleId: issue.ruleId,
			outcome: 'pass',
			subjectIds: ['occ-v2-original']
		});
		expect(evaluateVerification(request, report).status).toBe('fixed');
	});

	it('does not verify a passing rule on an authentication wall', () => {
		const report = current();
		report.issues = [{ ...issue, id: 'auth', category: 'auth', ruleId: 'auth-wall-detected' }];
		expect(
			evaluateVerification(buildVerificationRequest(original(), issue, 'verification-job'), report)
				.status
		).toBe('unverified');
	});

	it('allows fresh ZIP builds with a changed temporary origin and the same audited path', () => {
		const source = original();
		source.meta.scanContext!.inputType = 'zip';
		source.scanners[0]!.pages![0]!.finalUrl = 'http://old-extractor:8080/';
		const report = current();
		report.meta.scanContext!.inputType = 'zip';
		report.scanners[0]!.pages![0]!.finalUrl = 'http://fresh-extractor:8080/';
		expect(
			evaluateVerification(buildVerificationRequest(source, issue, 'verification-job'), report)
				.status
		).toBe('fixed');
	});

	it('keeps bounded local context without credentials or scanner options', () => {
		const values = new Map<string, string>();
		const storage = {
			getItem: (key: string) => values.get(key) ?? null,
			setItem: (key: string, value: string) => {
				values.set(key, value);
			}
		};
		const source = original();
		source.meta.scanContext!.scannerConfigurations!.axe = {
			password: 'never-store',
			token: 'credential'
		};
		for (let index = 0; index < 25; index++)
			expect(
				saveVerificationRequest(buildVerificationRequest(source, issue, `job-${index}`), storage)
			).toBe(true);
		const raw = storage.getItem(VERIFICATION_STORAGE_KEY)!;
		expect(raw).not.toContain('never-store');
		expect(raw).not.toContain('credential');
		expect(raw).not.toContain('scannerConfigurations');
		expect(JSON.parse(raw)).toHaveLength(20);
		expect(loadVerificationRequest('job-24', storage)?.sourceJobId).toBe('original-job');
		expect(loadVerificationRequest('job-0', storage)).toBeNull();
	});

	it.each([
		null,
		{ issue: 42 },
		{ issue: 'text', sourceContext: true },
		{ issue, sourceContext: {} }
	])('rejects malformed descriptors without throwing', (value) =>
		expect(parseVerificationRequest(value)).toBeNull()
	);

	it('requires original context and reports unavailable storage without submitting anything', () => {
		const legacy = original();
		delete legacy.meta.scanContext;
		expect(verificationUnavailableReason(legacy, issue)).toContain('original scan settings');
		expect(
			saveVerificationRequest(buildVerificationRequest(original(), issue, 'job'), {
				getItem: () => null,
				setItem: () => {
					throw new Error('quota');
				}
			})
		).toBe(false);
	});

	it('rejects primitive descriptor fields and cyclic values after the outer identity fields validate', () => {
		const request = buildVerificationRequest(original(), issue, 'verification-job');
		expect(parseVerificationRequest({ ...request, issue: 42 })).toBeNull();
		expect(parseVerificationRequest({ ...request, sourceContext: true })).toBeNull();
		const cyclic: Record<string, unknown> = { ...request };
		cyclic.self = cyclic;
		expect(parseVerificationRequest(cyclic)).toBeNull();
	});

	it('refuses settings that cannot be faithfully replayed by URL or ZIP intake', () => {
		const report = original();
		report.meta.scanContext!.verificationEligiblePages = { [pageId]: false };
		expect(verificationUnavailableReason(report, issue)).toContain(
			'custom viewport, wait or pre-scan actions'
		);
	});
});
