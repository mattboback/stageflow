import { describe, expect, it } from 'vitest';
import { loadAllScansFixture } from '../test/load-fixture';
import { buildImportedCliRun, parseCliReport } from './cli-report-import';
import { portableScanEnvelope } from './portable-audit';
import type { LocalProject } from './projects';

function fixture() {
	const report = loadAllScansFixture();
	report.meta.scanContext = {
		expectedPages: report.pages.map((page) => page.id),
		identityScope: 'path',
		scannerConfigFingerprints: Object.fromEntries(
			report.scanners.map((scanner) => [scanner.id, 'settings'])
		),
		scannerConfigurations: Object.fromEntries(report.scanners.map((scanner) => [scanner.id, {}])),
		verificationEligiblePages: Object.fromEntries(report.pages.map((page) => [page.id, true])),
		viewport: { width: 1280, height: 720 },
		purpose: 'full',
		authenticated: false,
		inputType: 'urls'
	};
	const project: LocalProject = {
		id: 'project',
		name: 'Website',
		schemaVersion: 1,
		createdAt: 'today',
		updatedAt: 'today',
		configuration: {
			urls: report.pages.map((page) => page.url),
			scanners: report.scanners.map((scanner) => ({ id: scanner.id, enabled: true })),
			browser: 'chromium',
			highlightStyle: 'solid',
			profile: report.meta.profile ?? 'production'
		}
	};
	return { report, project, envelope: portableScanEnvelope(report) };
}

describe('CLI report import', () => {
	it('round trips a complete CLI envelope into an archived run without remote screenshots', async () => {
		const { project, envelope } = fixture();
		const run = await buildImportedCliRun(
			project,
			parseCliReport(JSON.parse(JSON.stringify(envelope)))
		);
		expect(run.projectId).toBe(project.id);
		expect(run.report).toEqual(envelope.report);
		expect(run.screenshots).toEqual([]);
		expect(run.archived).toBe(true);
		expect(run.configFingerprint).toMatch(/^identity-v2:/);
	});

	it('rejects truncation, filtering, inconsistent job identity and malformed reports', () => {
		const { envelope } = fixture();
		for (const change of [
			{ filters: { ...envelope.filters, truncated: true } },
			{ filters: { ...envelope.filters, issues_total: 999 } },
			{ filters: { ...envelope.filters, categories: ['accessibility'] } },
			{ job: { id: 'wrong' } },
			{ report: { ...envelope.report, pages: [null] } }
		])
			expect(() => parseCliReport({ ...envelope, ...change })).toThrow();
	});

	it.each([
		'profile',
		'pages',
		'scanners',
		'config',
		'auth',
		'verification',
		'viewport',
		'actions'
	])('rejects incompatible %s before saving', async (change) => {
		const { project, report } = fixture();
		const context = report.meta.scanContext!;
		if (change === 'profile')
			report.meta.profile = project.configuration.profile === 'staging' ? 'production' : 'staging';
		if (change === 'pages') report.pages[0]!.url = 'https://different.example/';
		if (change === 'scanners') report.scanners.pop();
		if (change === 'config')
			context.scannerConfigurations![report.scanners[0]!.id] = { changed: true };
		if (change === 'auth') context.authenticated = true;
		if (change === 'verification') context.purpose = 'verification';
		if (change === 'viewport') context.viewport!.width = 400;
		if (change === 'actions') context.verificationEligiblePages![report.pages[0]!.id] = false;
		await expect(buildImportedCliRun(project, report)).rejects.toThrow();
	});

	it('retains partial scan evidence rather than turning it into a passing report', async () => {
		const { project, report } = fixture();
		report.scanners[0]!.status = 'failed';
		const run = await buildImportedCliRun(project, report);
		expect(run.report?.scanners[0]?.status).toBe('failed');
	});

	it('matches reordered URLs and config keys without changing saved settings', async () => {
		const { project, report } = fixture();
		project.configuration.urls.reverse();
		project.configuration.scanners[0]!.config = { a: 1, b: 2 };
		report.meta.scanContext!.scannerConfigurations![project.configuration.scanners[0]!.id] = {
			b: 2,
			a: 1
		};
		const before = structuredClone(project);
		await expect(buildImportedCliRun(project, report)).resolves.toBeDefined();
		expect(project).toEqual(before);
	});
});
