import { afterEach, describe, expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../test/load-fixture';
import {
	embedAuditJSON,
	parsePortableAudit,
	portableScanEnvelope,
	preparePortableAudit,
	renderPortableAuditHTML,
	type PortableAudit
} from './portable-audit';

const report = loadAllScansFixture();
function audit(): PortableAudit {
	return {
		schema: 'stageflow/portable-audit@v1',
		jobId: report.meta.jobId,
		createdAt: '2026-09-30T12:00:00Z',
		report,
		screenshots: [],
		missingArtifacts: [],
		reviews: {
			[report.meta.jobId]: {
				issue: { verdict: 'pass', note: 'Confirmed visually', at: '2026-09-30T12:00:00Z' }
			}
		}
	};
}

afterEach(() => vi.unstubAllGlobals());

describe('portable audit', () => {
	it('embeds untrusted scanner content without breaking out of script/style tags', () => {
		const value = { html: '</script><script>alert(1)</script>&\u2028\u2029' };
		const encoded = embedAuditJSON(value);
		expect(encoded).not.toContain('<');
		expect(JSON.parse(encoded)).toEqual(value);
		const html = renderPortableAuditHTML(
			{ ...audit(), reviews: { [report.meta.jobId]: { issue: { note: value.html, at: 'now' } } } },
			{ javascript: 'const text="</script>";', css: '.x:before{content:"</style>"}' }
		);
		const document = new DOMParser().parseFromString(html, 'text/html');
		expect(document.querySelectorAll('script')).toHaveLength(2);
		expect(document.querySelectorAll('style')).toHaveLength(1);
		expect(document.querySelectorAll('[src],link')).toHaveLength(0);
		expect(
			document.querySelector('meta[http-equiv="Content-Security-Policy"]')?.getAttribute('content')
		).toContain("connect-src 'none'");
		expect(
			parsePortableAudit(JSON.parse(document.getElementById('stageflow-audit')!.textContent))
				?.reviews[report.meta.jobId]?.issue?.note
		).toBe(value.html);
	});

	it('round trips raw report, reviews and embedded evidence; excludes secret configuration', () => {
		const input = {
			...audit(),
			screenshots: [
				{
					kind: 'page_overview' as const,
					artifact_id: 'image',
					scanner_id: 'axe',
					page_id: report.pages[0]!.id,
					url: 'data:image/webp;base64,AA=='
				}
			],
			configuration: {
				urls: ['https://example.com'],
				browser: 'chromium' as const,
				highlightStyle: 'solid' as const,
				scanners: [
					{
						id: 'axe',
						enabled: true,
						config: { threshold: 4.5, password: 'secret', auth: { token: 'credential' } }
					}
				]
			}
		};
		const parsed = parsePortableAudit(JSON.parse(JSON.stringify(input)));
		expect(parsed?.report).toEqual(report);
		expect(parsed?.reviews).toEqual(input.reviews);
		expect(parsed?.screenshots).toEqual(input.screenshots);
		expect(parsed?.configuration?.scanners[0]?.config).toEqual({ threshold: 4.5 });
		expect(portableScanEnvelope(report).report).toBe(report);
		expect(portableScanEnvelope(report).schema).toBe('stageflow-cli/report@v1');
	});

	it('rejects remote or executable media and malformed configuration', () => {
		const screenshot = {
			kind: 'page_overview',
			artifact_id: 'image',
			scanner_id: 'axe',
			page_id: 'page',
			url: 'https://example.com/image.webp'
		};
		expect(parsePortableAudit({ ...audit(), screenshots: [screenshot] })).toBeNull();
		expect(
			parsePortableAudit({
				...audit(),
				screenshots: [{ ...screenshot, url: 'data:image/svg+xml;base64,AA==' }]
			})
		).toBeNull();
		expect(
			parsePortableAudit({ ...audit(), configuration: { urls: [123], scanners: [] } })
		).toBeNull();
	});

	it('downloads bounded screenshot evidence and reports expired or oversized media explicitly', async () => {
		const screenshots = ['good', 'expired', 'large'].map((id) => ({
			kind: 'page_overview' as const,
			artifact_id: id,
			scanner_id: 'axe',
			page_id: report.pages[0]!.id,
			url: `https://example.com/${id}.webp`
		}));
		vi.stubGlobal(
			'fetch',
			vi.fn((url: URL) =>
				Promise.resolve(
					url.pathname.includes('expired')
						? new Response('expired', { status: 403 })
						: new Response(new Uint8Array([1, 2]), {
								headers: {
									'content-type': 'image/webp',
									...(url.pathname.includes('large') ? { 'content-length': '9000000' } : {})
								}
							})
				)
			)
		);
		const result = await preparePortableAudit({ jobId: report.meta.jobId, report, screenshots });
		expect(result.screenshots).toHaveLength(1);
		expect(result.screenshots[0]?.url).toBe('data:image/webp;base64,AQI=');
		expect(
			result.missingArtifacts.some(
				(missing) => missing.artifactId === 'expired' && missing.reason.includes('403')
			)
		).toBe(true);
		expect(
			result.missingArtifacts.some(
				(missing) => missing.artifactId === 'large' && missing.reason.includes('size limit')
			)
		).toBe(true);
		expect(result.report).toBe(report);
	});
});

it('keeps only the current and included baseline reviews in portable files', async () => {
	const baseline = { ...report, meta: { ...report.meta, jobId: 'included-baseline' } };
	const reviews = {
		[report.meta.jobId]: { issue: { at: 'today', note: 'Current review' } },
		'included-baseline': { issue: { at: 'yesterday', note: 'Baseline review' } },
		'other-private-audit': { issue: { at: 'yesterday', note: 'Unrelated confidential notes' } }
	};
	const result = await preparePortableAudit({
		jobId: report.meta.jobId,
		report,
		baseline,
		reviews,
		screenshots: []
	});
	expect(Object.keys(result.reviews)).toEqual([report.meta.jobId, 'included-baseline']);
	const html = renderPortableAuditHTML(
		{ ...result, reviews },
		{ javascript: 'console.log(1)', css: 'body{}' }
	);
	expect(html).not.toContain('Unrelated confidential notes');
	expect(html).toContain('Baseline review');
});

it('rejects inconsistent job identity and malformed base64 without opening a broken viewer', () => {
	expect(parsePortableAudit({ ...audit(), jobId: 'wrong-job' })).toBeNull();
	const screenshot = {
		kind: 'page_overview',
		artifact_id: 'shot',
		scanner_id: 'axe',
		page_id: 'page',
		url: 'data:image/png;base64,A==='
	};
	expect(parsePortableAudit({ ...audit(), screenshots: [screenshot] })).toBeNull();
	screenshot.url = 'data:image/png;base64,A';
	expect(parsePortableAudit({ ...audit(), screenshots: [screenshot] })).toBeNull();
});

it('rejects supplied malformed baselines and scanner configurations instead of dropping them', () => {
	for (const baseline of [null, {}, { ...report, issues: [null] }]) {
		expect(parsePortableAudit({ ...audit(), baseline })).toBeNull();
	}
	for (const scanners of [[null], [{ id: 'axe', enabled: true, config: [] }]]) {
		expect(
			parsePortableAudit({ ...audit(), configuration: { urls: ['https://example.com'], scanners } })
		).toBeNull();
	}
});

it('rejects malformed comparison findings before rendering imported audit data', () => {
	const comparison = {
		schema: 'stageflow/diff@v1',
		baseline: { jobId: 'before', totalIssues: 0 },
		current: { jobId: report.meta.jobId, totalIssues: 1 },
		delta: { newIssues: 1, fixedIssues: 0, unchangedIssues: 0 },
		new: [report.issues[0]],
		fixed: []
	};
	expect(parsePortableAudit({ ...audit(), comparison })?.comparison).toEqual(comparison);
	for (const patch of [
		{ new: [null] },
		{ fixed: [{}] },
		{ baseline: { jobId: 'before' } },
		{ delta: { newIssues: 'one', fixedIssues: 0, unchangedIssues: 0 } },
		{ skippedScanners: [null] }
	]) {
		expect(parsePortableAudit({ ...audit(), comparison: { ...comparison, ...patch } })).toBeNull();
	}
});
