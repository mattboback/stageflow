import { afterEach, expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../test/load-fixture';
import { importLocalProject } from './local-project-store';
import { PORTABLE_AUDIT_SCHEMA } from './portable-audit';

afterEach(() => vi.unstubAllGlobals());

it('rejects malformed imported reports before opening browser storage', async () => {
	const open = vi.fn(() => {
		throw new Error('Storage must not be touched');
	});
	vi.stubGlobal('indexedDB', { open });
	const report = loadAllScansFixture();
	const audit = {
		schema: PORTABLE_AUDIT_SCHEMA,
		jobId: report.meta.jobId,
		createdAt: 'today',
		report,
		screenshots: [],
		missingArtifacts: [],
		reviews: {}
	};
	for (const value of [
		{ ...audit, report: { ...report, pages: [null] } },
		{ ...audit, baseline: { ...report, scanners: [null] } },
		{ ...audit, configuration: { urls: [null], scanners: [] } }
	])
		await expect(importLocalProject(value)).rejects.toThrow();
	expect(open).not.toHaveBeenCalled();
});
